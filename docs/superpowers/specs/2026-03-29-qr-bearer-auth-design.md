# QR Pairing + Persistent Bearer Token Authentication Design

**Date:** 2026-03-29  
**Status:** Approved  
**Author:** API authentication redesign for QR-based mobile pairing

---

## Overview

Replace the current static header/basic-password gate with bearer-token authentication for all API routes. At server startup, Gitty prints a QR code. A mobile client scans it, submits the configured `masterPassword`, and receives a per-device opaque bearer token that persists across restarts until revoked.

This design enforces authentication for all client API calls while keeping pairing simple for mobile devices.

---

## Requirements

### Functional Requirements

1. **All API routes protected by bearer auth** except explicit public routes.
2. **Public routes limited to:**
   - `GET /health`
   - `/api/auth/pair/*` pairing endpoints
3. **Startup QR pairing**: server prints a QR payload at boot.
4. **QR + secret flow**: scanning QR is not enough; pairing must include valid `masterPassword`.
5. **Per-device tokens**: each client receives its own token/device record.
6. **Persistent tokens**: tokens survive server restarts.
7. **Per-device revocation**: revoke one token without affecting others.
8. **Opaque token format**: server-issued random bearer tokens with server-side validation.
9. **Fail-fast startup** if `masterPassword` is missing or empty.

### Non-Functional Requirements

1. No raw token persistence (store token hashes only).
2. Constant-time token hash comparison.
3. Clear route-level auth policy and deterministic status codes.
4. Minimal auth logging (no secrets, no token values in logs).

---

## Current-State Constraints

Current codebase facts this design must respect:

- `server/cmd/gittyd/main.go` currently applies `auth.PasswordGate(...)` as a global gate.
- `server/internal/auth/password.go` supports `X-Gitty-Master` and basic auth.
- Core API routes live in `server/internal/api/routes.go` under `/api`.
- Router tests currently expect unauthenticated access in `server/internal/api/routes_test.go` and will require updates once bearer gating is introduced.

---

## Architecture

### High-Level Components

1. **Pairing Session Service** (`internal/auth/pairing`)
- Creates short-lived, single-use pairing sessions for QR exchanges.
- Validates pairing sessions (`exists`, `not expired`, `not used`).

2. **Token Service + Store** (`internal/auth/tokens`)
- Issues high-entropy opaque tokens.
- Stores token hashes and device metadata on disk.
- Loads persisted token records on startup.
- Supports list/revoke operations by device ID.

3. **Bearer Middleware** (`internal/auth/bearer`)
- Validates `Authorization: Bearer <token>`.
- Rejects missing/invalid/revoked tokens with `401`.
- Passes authenticated device context downstream.

4. **Auth Handlers** (`internal/api/handlers/auth.go`)
- Pairing exchange endpoint.
- Device list and revoke endpoints.

### Route Protection Model

- **Public:**
  - `GET /health`
  - `POST /api/auth/pair/exchange`
  - Optional read endpoint for active pairing metadata, e.g. `GET /api/auth/pair/session`
- **Bearer-protected:** all remaining `/api/*` endpoints.

---

## Data Model

### Pairing Session (In-Memory)

```go
type PairSession struct {
    SessionID string
    CreatedAt time.Time
    ExpiresAt time.Time
    UsedAt    *time.Time
}
```

Notes:
- TTL default: 10 minutes.
- Single use: successful exchange sets `UsedAt`.
- Not persisted: new pairing session generated on each server boot.

### Token Record (Persisted)

```go
type DeviceTokenRecord struct {
    DeviceID     string     `json:"deviceId"`
    DeviceName   string     `json:"deviceName"`
    TokenHash    string     `json:"tokenHash"`
    CreatedAt    time.Time  `json:"createdAt"`
    LastUsedAt   *time.Time `json:"lastUsedAt,omitempty"`
    RevokedAt    *time.Time `json:"revokedAt,omitempty"`
}
```

Persisted envelope:

```go
type TokenStoreFile struct {
    Version int                 `json:"version"`
    Tokens  []DeviceTokenRecord `json:"tokens"`
}
```

Storage path:
- `~/.config/gitty/auth-tokens.json` (0600 file mode)

---

## API Contract

### 1) Pair Exchange

`POST /api/auth/pair/exchange` (public)

Request:

```json
{
  "sessionId": "ps_123",
  "masterPassword": "secret",
  "deviceName": "iPhone 15 Pro"
}
```

Success (`200`):

```json
{
  "accessToken": "gty_xxx",
  "tokenType": "Bearer",
  "deviceId": "dev_abc",
  "expiresAt": null
}
```

Errors:
- `401` invalid master password
- `410` pairing session expired or already used
- `404` pairing session not found
- `429` too many failed attempts (if rate limiter enabled)

### 2) List Devices

`GET /api/auth/devices` (bearer)

Success (`200`):

```json
{
  "devices": [
    {
      "deviceId": "dev_abc",
      "deviceName": "iPhone 15 Pro",
      "createdAt": "2026-03-29T12:00:00Z",
      "lastUsedAt": "2026-03-29T12:10:00Z",
      "revokedAt": null
    }
  ]
}
```

### 3) Revoke Device

`DELETE /api/auth/devices/{deviceId}` (bearer)

Responses:
- `204` revoked
- `404` device not found

---

## Startup and QR Behavior

At daemon startup:

1. Validate `masterPassword` exists and is non-empty.
2. Initialize persisted token store from disk.
3. Create one pairing session (`sessionId`, `expiresAt`).
4. Compose QR payload with:
   - `baseUrl` (network-reachable server URL)
   - `sessionId`
   - `expiresAt`
5. Print QR code (ASCII terminal) and plain-text fallback payload.

Example payload:

```json
{
  "baseUrl": "http://192.168.1.10:8083",
  "sessionId": "ps_123",
  "expiresAt": "2026-03-29T12:10:00Z"
}
```

---

## Security Design

1. **Password use boundary**
- `masterPassword` is used only during pairing exchange.
- Normal API traffic never uses `X-Gitty-Master` or basic auth.

2. **Token generation**
- Generate >=32 random bytes using `crypto/rand`.
- Encode as URL-safe string for `Authorization` header use.

3. **Token storage**
- Persist only token hash (`sha256(token)` or HMAC variant).
- Never log raw tokens.

4. **Comparison safety**
- Use constant-time compare for hash checks.

5. **Revocation behavior**
- Revocation takes effect immediately.
- Revoked token always fails with `401`.

6. **Failure semantics**
- Unauthorized requests return `401` without disclosing whether token exists.

---

## Error Handling

| Scenario | Status | Response |
|---|---:|---|
| Missing bearer header | 401 | `{"error":"missing bearer token"}` |
| Malformed bearer header | 401 | `{"error":"invalid bearer token"}` |
| Invalid/revoked token | 401 | `{"error":"invalid bearer token"}` |
| Invalid master password | 401 | `{"error":"invalid credentials"}` |
| Pair session not found | 404 | `{"error":"pair session not found"}` |
| Pair session expired/used | 410 | `{"error":"pair session unavailable"}` |
| Device not found on revoke | 404 | `{"error":"device not found"}` |

---

## Testing Strategy

### Unit Tests

1. Pair session lifecycle:
- create session
- expire session
- single-use enforcement

2. Token store:
- issue token + hash persist
- reload persisted file
- validate token
- revoke token
- revoked token rejection

3. Startup validation:
- boot fails when `masterPassword` missing

### Integration/API Tests

1. Public route access:
- `/health` reachable without bearer
- `/api/auth/pair/exchange` reachable without bearer

2. Protected route access:
- `/api/repos` returns `401` without bearer
- `/api/repos` succeeds with valid bearer

3. Pair exchange matrix:
- success case
- bad password
- unknown session
- expired session
- reused session

4. Device management:
- list devices
- revoke device
- revoked token denied on protected endpoint

---

## Migration Plan

1. Add bearer infrastructure and auth endpoints.
2. Apply bearer middleware to `/api/*` routes, exempt pair endpoints.
3. Remove `PasswordGate` from general API routing.
4. Keep `masterPassword` config as pairing secret only.
5. Update docs/OpenAPI/client usage notes.

---

## Out of Scope

1. Multi-user account model.
2. External IdP/OAuth.
3. Fine-grained authorization scopes/roles.
4. Token refresh protocol (tokens are long-lived and revocable).

---

## Acceptance Criteria

1. Server refuses to start without `masterPassword`.
2. Startup prints scan-ready QR payload.
3. Mobile pairing requires valid QR session + valid `masterPassword`.
4. Pairing returns per-device opaque bearer token.
5. Token remains valid after restart.
6. All `/api/*` routes reject missing/invalid bearer tokens.
7. Single device token revocation works immediately.
