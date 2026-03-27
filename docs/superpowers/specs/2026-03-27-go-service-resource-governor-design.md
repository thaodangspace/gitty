# Go Service Resource Governor - Design Specification

**Date:** 2026-03-27
**Status:** Approved

## Problem Statement

The Go backend currently has endpoint-level pagination and a few localized safeguards, but it does not have a consistent process-level resource governance model. Under heavy expensive workloads (diff/tokenization and commit history), the service can over-consume CPU and memory, reducing responsiveness for lightweight endpoints.

We need the service to self-enforce process limits and degrade expensive operations gracefully before the process becomes unstable.

## Goals

1. Enforce process-level caps in-service using `GOMEMLIMIT` semantics and `GOMAXPROCS`
2. Keep lightweight endpoints responsive when pressure is high
3. Degrade only expensive endpoints first (diff/tokenization and commit history)
4. Return deterministic overload behavior (`503` + `Retry-After`) instead of timing out unpredictably
5. Keep implementation incremental with minimal API surface change

## Non-Goals

1. Container or systemd-level enforcement (this phase is app-level enforcement)
2. Converting expensive endpoints into asynchronous job workflows
3. Dynamic/ML-style adaptive controllers in this phase
4. Full observability platform rollout (basic counters/logging only)

## Scope

### Expensive Endpoints (degraded first)

- `GET /api/repos/{id}/commits`
- `GET /api/repos/{id}/diff/*`
- `GET /api/repos/{id}/diff/tokenized/*`
- `GET /api/repos/{id}/diff/commit/tokenized`
- `GET /api/repos/{id}/diff/commit/{hash}/files/*`

### Protected Endpoints (must stay responsive)

- `GET /health`
- `GET /api/repos/{id}/status`
- Other non-expensive routes continue to bypass expensive admission control

## Design Overview

The solution adds a resource governor layer that combines runtime caps, pressure monitoring, and expensive-route admission control:

1. Configure runtime caps at startup:
- default `memory_limit = 1GiB`
- default `gomaxprocs = 2`

2. Monitor in-process memory pressure and expose mode:
- `normal`
- `degraded`

3. Gate expensive requests through bounded concurrency admission:
- admit when mode is `normal` and token is available
- reject with `503` when degraded or saturated

4. Keep lightweight endpoints available while expensive routes degrade first.

## Architecture

### 1. Runtime Cap Bootstrap

Add startup initialization in server boot path to apply runtime caps from config/env:

- `debug.SetMemoryLimit(memoryLimitBytes)`
- `runtime.GOMAXPROCS(gomaxprocs)`

Defaults:

- `memoryLimitBytes = 1GiB`
- `gomaxprocs = 2`

Validation:

- Memory limit must be `> 0`
- `gomaxprocs` must be `>= 1`
- Invalid values fail fast at startup with clear error

### 2. Governor Component

Introduce `server/internal/resources` package with:

- `Governor` (public interface)
- `PressureMonitor` (periodic sampler)
- `AdmissionController` (bounded semaphore)

Representative interface:

```go
type Mode string

const (
    ModeNormal   Mode = "normal"
    ModeDegraded Mode = "degraded"
)

type Governor interface {
    Mode() Mode
    TryEnterExpensive() (release func(), ok bool, reason string)
}
```

### 3. Pressure Monitor

Sample runtime memory stats on an interval (for example every 250-500ms), compute pressure ratio against configured memory limit, and apply hysteresis:

- Enter degraded when `pressure >= 0.85`
- Exit degraded when `pressure <= 0.70`

If pressure sampling fails, governor falls back to limiter-only behavior and logs the event.

### 4. Expensive Admission Control

Maintain an expensive-work semaphore (`max_expensive_inflight`, configurable with conservative default such as `2`).

Decision order for expensive endpoints:

1. If mode is `degraded` -> reject (`reason=degraded_mode`)
2. Else attempt semaphore acquire:
- success -> continue handler, release on exit
- fail -> reject (`reason=expensive_limit_reached`)

## Request Flow

1. Request reaches route handler
2. If route is not expensive: execute as today
3. If route is expensive: governor pre-check
4. If rejected: return `503` + JSON error payload + `Retry-After`
5. If admitted: run existing Git service operation unchanged
6. Release token after response completion
7. Pressure monitor updates mode in background

## Error Handling and Client Contract

For expensive route rejection:

- Status: `503 Service Unavailable`
- Header: `Retry-After: 3` (configurable)
- Body:

```json
{
  "error": "service_degraded",
  "reason": "degraded_mode"
}
```

or

```json
{
  "error": "service_busy",
  "reason": "expensive_limit_reached"
}
```

No crash behavior is introduced by design in this phase.

## Configuration

Add governor settings to app config (env and/or config file-backed):

- `resourceGovernor.enabled` (default `true`)
- `resourceGovernor.memoryLimitBytes` (default `1073741824`)
- `resourceGovernor.gomaxprocs` (default `2`)
- `resourceGovernor.maxExpensiveInflight` (default `2`)
- `resourceGovernor.sampleIntervalMs` (default `500`)
- `resourceGovernor.degradeHighWatermark` (default `0.85`)
- `resourceGovernor.degradeLowWatermark` (default `0.70`)
- `resourceGovernor.retryAfterSeconds` (default `3`)

Validation rules:

- `degradeLowWatermark < degradeHighWatermark`
- both in range `(0, 1]`
- numeric values positive where applicable

## Observability

Minimum required signals:

- Counter: `governor_rejections_total{reason}`
- Gauge: `expensive_inflight`
- Current mode in logs/events: `normal|degraded`

At minimum, structured logs must include: route, reason, inflight, mode, and current pressure ratio.

## Testing Strategy

### Unit Tests (`server/internal/resources`)

1. Hysteresis transitions:
- `normal -> degraded` at high watermark
- `degraded -> normal` at low watermark
- no flapping between thresholds

2. Admission controller behavior:
- admits when capacity is available
- rejects when capacity is exhausted
- rejects while degraded regardless of free tokens

3. Fallback behavior:
- pressure sampling failure does not panic
- limiter-only path remains functional

### Handler-Level Tests

For expensive endpoints:

- returns `503` when governor is degraded
- returns `503` when expensive inflight is saturated
- succeeds when mode is normal and token is available

For non-expensive endpoints:

- behavior unchanged under degraded mode

### Startup/Config Tests

- defaults applied when unset
- valid overrides honored
- invalid values fail startup with explicit errors

### Integration Smoke

- run with low memory limit in test setup
- drive expensive routes
- verify degraded behavior on expensive endpoints while `/health` stays `200`

## Rollout Plan

1. Add governor package and unit tests
2. Wire startup runtime caps and config validation
3. Add route classification + expensive route gate wrappers
4. Add rejection payload/header behavior
5. Add logging/counters
6. Run full backend test suite and targeted smoke checks

## Risks and Mitigations

1. Risk: thresholds too aggressive reduce throughput
- Mitigation: expose thresholds and inflight limits in config

2. Risk: route classification drift as new expensive endpoints are added
- Mitigation: centralize expensive-route matcher and add tests per route

3. Risk: monitor overhead or noisy logs
- Mitigation: lightweight sampling interval and rate-limited logs

## Open Questions

1. Should `/api/repos/{id}/diff/commit/tokenized` and `/api/repos/{id}/diff/*` share one bucket or separate buckets?
- Current design: single bucket for simplicity

2. Should retry duration be static or pressure-dependent?
- Current design: static `Retry-After` for deterministic behavior
