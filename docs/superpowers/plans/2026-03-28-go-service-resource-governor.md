# Go Service Resource Governor Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add in-service CPU/memory guardrails with graceful degradation for expensive repository endpoints while keeping lightweight routes responsive.

**Architecture:** Introduce a small `resources` package that owns runtime caps (`SetMemoryLimit`, `GOMAXPROCS`), pressure-mode transitions (normal/degraded), and expensive-work admission control via a bounded semaphore. Wire this governor into repository handlers for commit-history and diff/tokenization routes so overload returns deterministic `503` responses with `Retry-After`.

**Tech Stack:** Go 1.23, chi router/handlers, standard library (`runtime`, `runtime/debug`, `sync/atomic`, `time`), Go test (`go test`)

---

## Implementation Notes

- Follow `@superpowers:test-driven-development` for each task.
- Keep scope aligned with spec: no async job queue, no adaptive controller.
- Prefer small commits at task boundaries.

## File Structure Map

### Create
- `server/internal/resources/config.go` - normalized governor settings + defaults + validation.
- `server/internal/resources/runtime.go` - runtime cap application (`SetMemoryLimit`, `GOMAXPROCS`).
- `server/internal/resources/governor.go` - mode state machine + expensive admission control.
- `server/internal/resources/governor_test.go` - unit coverage for mode transitions and admission behavior.
- `server/internal/resources/runtime_test.go` - unit coverage for config validation and cap application return values.

### Modify
- `server/internal/config/config.go` - add `ResourceGovernor` config section and validation/default plumbing.
- `server/internal/config/config_test.go` - add tests for governor config load/validation.
- `server/cmd/gittyd/main.go` - apply runtime caps on startup from effective config.
- `server/internal/api/handlers/repository.go` - add governor field, expensive-route gating helper, and `503` response contract.
- `server/internal/api/handlers/repository_test.go` - add expensive-route degradation/saturation tests and non-expensive-route safety checks.
- `server/internal/api/openapi.yml` - document `503` for guarded expensive routes (commit history + diff family).

---

### Task 1: Add Config Surface For Resource Governor

**Files:**
- Modify: `server/internal/config/config.go`
- Test: `server/internal/config/config_test.go`

- [ ] **Step 1: Write failing config tests for defaults and validation**

Add tests:
- `TestLoadConfigDefaultsResourceGovernor`
- `TestLoadConfigWithResourceGovernorOverrides`
- `TestLoadConfigInvalidResourceGovernorWatermarks`

Example test expectations:
```go
if cfg.ResourceGovernor.MemoryLimitBytes != 1073741824 {
    t.Fatalf("expected default memory limit 1GiB, got %d", cfg.ResourceGovernor.MemoryLimitBytes)
}
if cfg.ResourceGovernor.DegradeLowWatermark >= cfg.ResourceGovernor.DegradeHighWatermark {
    t.Fatalf("expected low watermark < high watermark")
}
```

- [ ] **Step 2: Run config tests to verify failure**

Run: `go test ./server/internal/config -run 'TestLoadConfig.*ResourceGovernor' -count=1`
Expected: FAIL with missing `ResourceGovernor` fields/types.

- [ ] **Step 3: Implement config model + defaults + validation**

Add to `Config`:
```go
type ResourceGovernorConfig struct {
    Enabled              bool    `json:"enabled"`
    MemoryLimitBytes     int64   `json:"memoryLimitBytes"`
    GOMAXPROCS           int     `json:"gomaxprocs"`
    MaxExpensiveInflight int     `json:"maxExpensiveInflight"`
    SampleIntervalMs     int     `json:"sampleIntervalMs"`
    DegradeHighWatermark float64 `json:"degradeHighWatermark"`
    DegradeLowWatermark  float64 `json:"degradeLowWatermark"`
    RetryAfterSeconds    int     `json:"retryAfterSeconds"`
}
```

Add effective-default function and validation:
- defaults: `1GiB`, `2`, `2`, `500ms`, `0.85`, `0.70`, `3`
- enforce `low < high`, both in `(0,1]`, positive numeric values.

- [ ] **Step 4: Re-run config tests**

Run: `go test ./server/internal/config -run 'TestLoadConfig.*ResourceGovernor' -count=1`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add server/internal/config/config.go server/internal/config/config_test.go
git commit -m "feat: add resource governor configuration"
```

### Task 2: Implement Runtime Cap Application Helpers

**Files:**
- Create: `server/internal/resources/runtime.go`
- Create: `server/internal/resources/runtime_test.go`
- Modify: `server/cmd/gittyd/main.go`

- [ ] **Step 1: Write failing runtime helper tests**

Add tests for:
- config with invalid values returns error
- valid config returns applied values struct

Example API target:
```go
type AppliedCaps struct {
    MemoryLimitBytes int64
    GOMAXPROCS       int
}

func ApplyRuntimeCaps(cfg Config) (AppliedCaps, error)
```

- [ ] **Step 2: Run resource package tests to verify failure**

Run: `go test ./server/internal/resources -run 'TestApplyRuntimeCaps' -count=1`
Expected: FAIL because package/functions do not exist yet.

- [ ] **Step 3: Implement `runtime.go` and wire startup call**

Implementation shape:
```go
func ApplyRuntimeCaps(cfg Config) (AppliedCaps, error) {
    if err := cfg.Validate(); err != nil {
        return AppliedCaps{}, err
    }
    debug.SetMemoryLimit(cfg.MemoryLimitBytes)
    runtime.GOMAXPROCS(cfg.GOMAXPROCS)
    return AppliedCaps{MemoryLimitBytes: cfg.MemoryLimitBytes, GOMAXPROCS: cfg.GOMAXPROCS}, nil
}
```

In `main.go`, after config load:
```go
caps, err := resources.ApplyRuntimeCaps(resources.FromAppConfig(cfg))
if err != nil { log.Fatalf("Invalid resource governor config: %v", err) }
log.Printf("Resource caps applied: memory=%d gomaxprocs=%d", caps.MemoryLimitBytes, caps.GOMAXPROCS)
```

- [ ] **Step 4: Re-run targeted tests**

Run: `go test ./server/internal/resources ./server/cmd/gittyd -count=1`
Expected: PASS (if `main` package has no tests, command still exits 0 for package compile).

- [ ] **Step 5: Commit**

```bash
git add server/internal/resources/runtime.go server/internal/resources/runtime_test.go server/cmd/gittyd/main.go
git commit -m "feat: apply runtime memory and cpu caps at startup"
```

### Task 3: Build Governor Core (Mode + Admission)

**Files:**
- Create: `server/internal/resources/config.go`
- Create: `server/internal/resources/governor.go`
- Create: `server/internal/resources/governor_test.go`

- [ ] **Step 1: Write failing governor unit tests**

Add tests:
- `TestGovernor_EntersDegradedAtHighWatermark`
- `TestGovernor_ExitsDegradedAtLowWatermark`
- `TestGovernor_RejectsWhenDegraded`
- `TestGovernor_RejectsWhenExpensiveInflightSaturated`
- `TestGovernor_AdmitsAndReleasesExpensiveToken`

Example assertion:
```go
release, ok, reason := g.TryEnterExpensive()
if !ok || reason != "" { t.Fatalf("expected admission") }
release()
```

- [ ] **Step 2: Run governor tests to verify failure**

Run: `go test ./server/internal/resources -run 'TestGovernor_' -count=1`
Expected: FAIL with missing governor symbols.

- [ ] **Step 3: Implement governor and defaults**

Add core structures:
```go
type Mode string
const (
    ModeNormal Mode = "normal"
    ModeDegraded Mode = "degraded"
)

type Decision struct {
    Allowed bool
    Reason  string // degraded_mode | expensive_limit_reached
    Release func()
}
```

Key behaviors:
- track mode atomically
- `UpdatePressure(ratio float64)` applies hysteresis (`>= high` degrade, `<= low` recover)
- expensive semaphore enforces `MaxExpensiveInflight`
- if degraded, reject immediately with `degraded_mode`

- [ ] **Step 4: Re-run governor tests**

Run: `go test ./server/internal/resources -run 'TestGovernor_' -count=1`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add server/internal/resources/config.go server/internal/resources/governor.go server/internal/resources/governor_test.go
git commit -m "feat: add resource governor mode and admission control"
```

### Task 4: Gate Expensive Repository Endpoints

**Files:**
- Modify: `server/internal/api/handlers/repository.go`
- Test: `server/internal/api/handlers/repository_test.go`

- [ ] **Step 1: Write failing handler tests for overload behavior**

Add tests:
- `TestGetCommitHistory_Returns503WhenDegraded`
- `TestHandleTokenizedFileDiff_Returns503WhenSaturated`
- `TestGetRepositoryStatus_NotBlockedWhenGovernorDegraded`

Expected response contract:
- status `503`
- header `Retry-After: 3`
- JSON reason (`degraded_mode` or `expensive_limit_reached`)

- [ ] **Step 2: Run targeted handler tests to verify failure**

Run: `go test ./server/internal/api/handlers -run 'TestGetCommitHistory_Returns503WhenDegraded|TestHandleTokenizedFileDiff_Returns503WhenSaturated|TestGetRepositoryStatus_NotBlockedWhenGovernorDegraded' -count=1`
Expected: FAIL because governor integration does not exist yet.

- [ ] **Step 3: Implement handler integration**

In `RepositoryHandler` add:
- `governor resources.Governor`
- helper:
```go
func (h *RepositoryHandler) enterExpensiveOrReject(w http.ResponseWriter) (release func(), ok bool)
```

Helper behavior:
- call `TryEnterExpensive`
- if rejected, write 503 JSON + retry header and return `ok=false`

Apply helper to expensive handlers only:
- `GetCommitHistory`
- `GetFileDiff`
- `HandleTokenizedFileDiff`
- `HandleTokenizedCommitDiff`
- `HandleCommitFileDiff`

- [ ] **Step 4: Re-run handler tests**

Run: `go test ./server/internal/api/handlers -run 'TestGetCommitHistory_Returns503WhenDegraded|TestHandleTokenizedFileDiff_Returns503WhenSaturated|TestGetRepositoryStatus_NotBlockedWhenGovernorDegraded' -count=1`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add server/internal/api/handlers/repository.go server/internal/api/handlers/repository_test.go
git commit -m "feat: degrade expensive repository endpoints under pressure"
```

### Task 5: Add Pressure Sampling Loop + Logging Signals

**Files:**
- Modify: `server/internal/api/handlers/repository.go`
- Test: `server/internal/api/handlers/repository_test.go`

- [ ] **Step 1: Write failing tests for monitor-driven mode changes**

Add test for monitor tick update path (can use injected sampler function):
- pressure above high watermark sets degraded
- pressure below low watermark recovers

- [ ] **Step 2: Run targeted tests to verify failure**

Run: `go test ./server/internal/api/handlers -run 'TestGovernorPressureSampling' -count=1`
Expected: FAIL with missing monitor hooks.

- [ ] **Step 3: Implement monitor loop + structured logs**

Implementation shape in handler init (or governor constructor):
```go
go func() {
    ticker := time.NewTicker(sampleInterval)
    defer ticker.Stop()
    for range ticker.C {
        ratio, err := pressureSampler()
        if err != nil { log.Printf("resource_governor sampler_error=%v", err); continue }
        prev, next := g.UpdatePressure(ratio)
        if prev != next {
            log.Printf("resource_governor mode=%s pressure=%.3f", next, ratio)
        }
    }
}()
```

Also log rejections with reason + route.

- [ ] **Step 4: Re-run target tests**

Run: `go test ./server/internal/api/handlers ./server/internal/resources -run 'TestGovernorPressureSampling|TestGovernor_' -count=1`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add server/internal/api/handlers/repository.go server/internal/api/handlers/repository_test.go
git commit -m "feat: add pressure sampling and governor logs"
```

### Task 6: Document API Behavior And Run Full Verification

**Files:**
- Modify: `server/internal/api/openapi.yml`

- [ ] **Step 1: Update OpenAPI for guarded expensive endpoints**

Add `503` response documentation for:
- commit history endpoint
- diff/tokenized endpoints
- commit file diff endpoint

Example schema snippet:
```yaml
503:
  description: Service degraded or busy
  headers:
    Retry-After:
      schema: { type: integer }
  content:
    application/json:
      schema:
        type: object
        properties:
          error: { type: string }
          reason: { type: string, enum: [degraded_mode, expensive_limit_reached] }
```

- [ ] **Step 2: Run package-level verification**

Run: `go test ./server/internal/resources ./server/internal/config ./server/internal/api/handlers ./server/internal/api -count=1`
Expected: PASS

- [ ] **Step 3: Run full backend verification**

Run: `go test ./server/... -count=1`
Expected: PASS (or capture unrelated pre-existing failures explicitly).

- [ ] **Step 4: Commit**

```bash
git add server/internal/api/openapi.yml
git commit -m "docs: document resource governor 503 responses"
```

## Final Verification Checklist

- [ ] `go test ./server/internal/resources -count=1`
- [ ] `go test ./server/internal/config -count=1`
- [ ] `go test ./server/internal/api/handlers -count=1`
- [ ] `go test ./server/... -count=1`
- [ ] Manual smoke: start server and verify `/health` stays `200` while expensive endpoints return `503` when governor is forced degraded.
