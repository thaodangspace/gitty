# QR Bearer Auth Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add QR-bootstrapped, per-device, persistent opaque bearer-token authentication so all `/api/*` endpoints require bearer auth except pairing endpoints and `/health`.

**Architecture:** Introduce a dedicated auth domain in `server/internal/auth` with three focused units: pairing session manager (ephemeral, single-use), token store (persisted hashed device tokens), and bearer middleware (request protection). Wire these through new auth handlers and route groups so pairing remains public while all repo/filesystem endpoints move behind `Authorization: Bearer`.

**Tech Stack:** Go 1.23, chi router, `crypto/rand`, `crypto/sha256`, JSON file persistence, optional `github.com/skip2/go-qrcode` for ASCII/terminal QR rendering.

---

## Scope Guardrails

- Keep scope to single-user, device-token authentication only.
- Do not introduce OAuth/JWT/refresh-token logic.
- Do not refactor unrelated repository/filesystem features.
- Keep legacy `masterPassword` as pairing secret only; do not accept it as normal API credential.

---

## File Structure

### Files to Create

1. `server/internal/auth/pairing.go`
- Pairing session model and manager (create/validate/mark-used)

2. `server/internal/auth/pairing_test.go`
- Unit tests for pairing session lifecycle and expiration

3. `server/internal/auth/token_store.go`
- Device token issuance, token hashing, disk persistence, revoke/list/validate

4. `server/internal/auth/token_store_test.go`
- Unit tests for persistence, validation, revocation, restart reload

5. `server/internal/auth/bearer.go`
- Bearer token extraction middleware and authenticated context helpers

6. `server/internal/auth/bearer_test.go`
- Middleware tests for missing/malformed/valid/revoked tokens

7. `server/internal/auth/qr_payload.go`
- Startup QR payload builder + encoder helper

8. `server/internal/auth/qr_payload_test.go`
- Tests for payload format and validation

9. `server/internal/api/handlers/auth.go`
- Pair exchange handler, list devices handler, revoke device handler

10. `server/internal/api/handlers/auth_test.go`
- Handler-level tests for auth endpoint behavior

### Files to Modify

1. `server/cmd/gittyd/main.go`
- Fail fast if `masterPassword` absent
- Initialize pairing manager + token store
- Print startup QR payload/ASCII QR
- Replace global `PasswordGate` API protection with bearer middleware strategy

2. `server/internal/api/routes.go`
- Register public auth pairing routes
- Apply bearer middleware to protected `/api` routes
- Inject auth dependencies into handlers

3. `server/internal/api/routes_test.go`
- Update router setup to account for bearer-protected endpoints
- Add route-level access tests (public vs protected)

4. `server/internal/api/openapi.yml`
- Add bearer security scheme and auth endpoints
- Mark protected endpoints with bearer security requirement

5. `server/internal/config/config.go`
- Add strict startup helper for required master password (or equivalent explicit guard)

6. `server/internal/config/config_test.go`
- Add test coverage for strict startup requirement helper

7. `server/README.md`
- Document QR pairing flow, bearer usage, and device revoke endpoints

8. `server/go.mod` and `server/go.sum` (only if QR lib added)

---

## Tasks

### Task 1: Pairing Session Manager (Ephemeral QR Session)

**Files:**
- Create: `server/internal/auth/pairing.go`
- Create: `server/internal/auth/pairing_test.go`
- Test: `server/internal/auth/pairing_test.go`

- [ ] **Step 1: Write failing pairing tests**

```go
func TestPairingManager_CreateAndValidate(t *testing.T) {
    mgr := NewPairingManager(10 * time.Minute)

    sess, err := mgr.CreateSession()
    if err != nil {
        t.Fatalf("create session: %v", err)
    }

    got, err := mgr.Validate(sess.SessionID)
    if err != nil {
        t.Fatalf("validate session: %v", err)
    }
    if got.SessionID != sess.SessionID {
        t.Fatalf("session id mismatch")
    }
}

func TestPairingManager_SingleUse(t *testing.T) {
    mgr := NewPairingManager(10 * time.Minute)
    sess, _ := mgr.CreateSession()

    if err := mgr.MarkUsed(sess.SessionID); err != nil {
        t.Fatalf("mark used: %v", err)
    }

    if _, err := mgr.Validate(sess.SessionID); !errors.Is(err, ErrPairSessionUnavailable) {
        t.Fatalf("expected ErrPairSessionUnavailable, got %v", err)
    }
}
```

- [ ] **Step 2: Run test to verify failure**

Run: `cd server && go test ./internal/auth -run PairingManager -v`  
Expected: compile errors (`undefined: NewPairingManager`, etc.)

- [ ] **Step 3: Implement minimal pairing manager**

```go
type PairSession struct {
    SessionID string
    CreatedAt time.Time
    ExpiresAt time.Time
    UsedAt    *time.Time
}

func (m *PairingManager) CreateSession() (*PairSession, error)
func (m *PairingManager) Validate(sessionID string) (*PairSession, error)
func (m *PairingManager) MarkUsed(sessionID string) error
```

Behavior:
- Session ID generated with crypto-random bytes.
- Validate rejects missing, expired, or used session with `ErrPairSessionUnavailable`.

- [ ] **Step 4: Run tests to verify pass**

Run: `cd server && go test ./internal/auth -run PairingManager -v`  
Expected: PASS for pairing tests

- [ ] **Step 5: Commit pairing unit**

```bash
git add server/internal/auth/pairing.go server/internal/auth/pairing_test.go
git commit -m "feat(auth): add ephemeral pairing session manager"
```

---

### Task 2: Persistent Device Token Store (Opaque Hash-Backed)

**Files:**
- Create: `server/internal/auth/token_store.go`
- Create: `server/internal/auth/token_store_test.go`
- Test: `server/internal/auth/token_store_test.go`

- [ ] **Step 1: Write failing token-store tests**

```go
func TestTokenStore_IssueValidateRevoke(t *testing.T) {
    dir := t.TempDir()
    store, err := NewTokenStore(filepath.Join(dir, "auth-tokens.json"))
    if err != nil {
        t.Fatalf("new token store: %v", err)
    }

    token, device, err := store.IssueToken("iPhone")
    if err != nil {
        t.Fatalf("issue token: %v", err)
    }
    if token == "" || device.DeviceID == "" {
        t.Fatalf("expected issued token and device id")
    }

    got, ok := store.Validate(token)
    if !ok || got.DeviceID != device.DeviceID {
        t.Fatalf("validate failed")
    }

    if err := store.Revoke(device.DeviceID); err != nil {
        t.Fatalf("revoke failed: %v", err)
    }

    if _, ok := store.Validate(token); ok {
        t.Fatalf("expected revoked token to fail validation")
    }
}

func TestTokenStore_PersistsAcrossRestart(t *testing.T) {
    path := filepath.Join(t.TempDir(), "auth-tokens.json")

    store1, _ := NewTokenStore(path)
    token, device, _ := store1.IssueToken("Pixel")

    store2, _ := NewTokenStore(path)
    got, ok := store2.Validate(token)
    if !ok || got.DeviceID != device.DeviceID {
        t.Fatalf("expected token to survive restart")
    }
}
```

- [ ] **Step 2: Run test to verify failure**

Run: `cd server && go test ./internal/auth -run TokenStore -v`  
Expected: compile errors (`undefined: NewTokenStore`, etc.)

- [ ] **Step 3: Implement token store with file persistence**

```go
type DeviceTokenRecord struct {
    DeviceID   string     `json:"deviceId"`
    DeviceName string     `json:"deviceName"`
    TokenHash  string     `json:"tokenHash"`
    CreatedAt  time.Time  `json:"createdAt"`
    LastUsedAt *time.Time `json:"lastUsedAt,omitempty"`
    RevokedAt  *time.Time `json:"revokedAt,omitempty"`
}

func (s *TokenStore) IssueToken(deviceName string) (rawToken string, rec DeviceTokenRecord, err error)
func (s *TokenStore) Validate(rawToken string) (DeviceTokenRecord, bool)
func (s *TokenStore) List() []DeviceTokenRecord
func (s *TokenStore) Revoke(deviceID string) error
```

Implementation requirements:
- Hash token before persist (never persist raw token).
- Persist file atomically (write temp + rename).
- Store file mode `0600`.
- Use mutex for thread safety.

- [ ] **Step 4: Run tests to verify pass**

Run: `cd server && go test ./internal/auth -run TokenStore -v`  
Expected: PASS for token store tests

- [ ] **Step 5: Commit token-store unit**

```bash
git add server/internal/auth/token_store.go server/internal/auth/token_store_test.go
git commit -m "feat(auth): add persistent per-device token store"
```

---

### Task 3: Bearer Middleware and Auth Context

**Files:**
- Create: `server/internal/auth/bearer.go`
- Create: `server/internal/auth/bearer_test.go`
- Test: `server/internal/auth/bearer_test.go`

- [ ] **Step 1: Write failing middleware tests**

```go
func TestBearerGate_RejectsMissingHeader(t *testing.T) {
    gate := BearerGate(fakeValidator{})
    rr := httptest.NewRecorder()
    req := httptest.NewRequest(http.MethodGet, "/api/repos", nil)

    gate(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
        t.Fatal("handler should not run")
    })).ServeHTTP(rr, req)

    if rr.Code != http.StatusUnauthorized {
        t.Fatalf("expected 401, got %d", rr.Code)
    }
}

func TestBearerGate_AllowsValidToken(t *testing.T) {
    gate := BearerGate(fakeValidator{allow: true})
    rr := httptest.NewRecorder()
    req := httptest.NewRequest(http.MethodGet, "/api/repos", nil)
    req.Header.Set("Authorization", "Bearer valid")

    called := false
    gate(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
        called = true
    })).ServeHTTP(rr, req)

    if !called {
        t.Fatal("expected downstream handler to run")
    }
}
```

- [ ] **Step 2: Run test to verify failure**

Run: `cd server && go test ./internal/auth -run BearerGate -v`  
Expected: compile errors (`undefined: BearerGate`)

- [ ] **Step 3: Implement bearer middleware**

```go
func BearerGate(validator TokenValidator) func(http.Handler) http.Handler
func DeviceFromContext(ctx context.Context) (DeviceTokenRecord, bool)
```

Rules:
- Accept only `Authorization: Bearer <token>`.
- Reject empty/malformed token with `401`.
- Validate against token store.
- Attach device record into request context.

- [ ] **Step 4: Run tests to verify pass**

Run: `cd server && go test ./internal/auth -run BearerGate -v`  
Expected: PASS

- [ ] **Step 5: Commit middleware unit**

```bash
git add server/internal/auth/bearer.go server/internal/auth/bearer_test.go
git commit -m "feat(auth): add bearer middleware with context device binding"
```

---

### Task 4: Auth HTTP Handlers (Pair Exchange + Device Management)

**Files:**
- Create: `server/internal/api/handlers/auth.go`
- Create: `server/internal/api/handlers/auth_test.go`
- Modify: `server/internal/api/handlers/repository.go` (only if shared response helpers required)
- Test: `server/internal/api/handlers/auth_test.go`

- [ ] **Step 1: Write failing handler tests**

```go
func TestAuthHandler_PairExchangeSuccess(t *testing.T) {
    h := newTestAuthHandler(t)

    body := strings.NewReader(`{"sessionId":"s1","masterPassword":"secret","deviceName":"iPhone"}`)
    req := httptest.NewRequest(http.MethodPost, "/api/auth/pair/exchange", body)
    rr := httptest.NewRecorder()

    h.PairExchange(rr, req)

    if rr.Code != http.StatusOK {
        t.Fatalf("expected 200, got %d", rr.Code)
    }
}

func TestAuthHandler_RevokeDevice(t *testing.T) {
    h := newTestAuthHandler(t)
    req := httptest.NewRequest(http.MethodDelete, "/api/auth/devices/dev_1", nil)
    rr := httptest.NewRecorder()

    h.RevokeDevice(rr, req)

    if rr.Code != http.StatusNoContent {
        t.Fatalf("expected 204, got %d", rr.Code)
    }
}
```

- [ ] **Step 2: Run test to verify failure**

Run: `cd server && go test ./internal/api/handlers -run AuthHandler -v`  
Expected: compile errors (`undefined: newTestAuthHandler`, missing handler methods)

- [ ] **Step 3: Implement auth handlers**

Required handler methods:
- `PairExchange(w, r)`
- `ListDevices(w, r)`
- `RevokeDevice(w, r)`

Required behavior:
- Validate JSON payload and required fields.
- Check `masterPassword` via constant-time compare helper.
- Mark pair session used on successful exchange.
- Return raw bearer token once (response only).
- Map errors to status codes from spec.

- [ ] **Step 4: Run tests to verify pass**

Run: `cd server && go test ./internal/api/handlers -run AuthHandler -v`  
Expected: PASS

- [ ] **Step 5: Commit handler unit**

```bash
git add server/internal/api/handlers/auth.go server/internal/api/handlers/auth_test.go
git commit -m "feat(api): add auth pairing and device management handlers"
```

---

### Task 5: Route Wiring and Server Bootstrap

**Files:**
- Modify: `server/internal/api/routes.go`
- Modify: `server/internal/api/routes_test.go`
- Modify: `server/cmd/gittyd/main.go`
- Modify: `server/internal/config/config.go`
- Modify: `server/internal/config/config_test.go`
- Test: `server/internal/api/routes_test.go`, `server/internal/config/config_test.go`

- [ ] **Step 1: Write/extend failing route and config tests**

Add tests for:
- `/health` open without bearer.
- `/api/auth/pair/exchange` open without bearer.
- `/api/repos` returns `401` without bearer.
- Startup guard helper errors when `masterPassword` missing.

- [ ] **Step 2: Run tests to verify failure**

Run:
- `cd server && go test ./internal/api -run NewRouter -v`
- `cd server && go test ./internal/config -run MasterPassword -v`

Expected: failures reflecting missing auth wiring and strict startup guard.

- [ ] **Step 3: Implement route groups and startup init**

Implementation checklist:
- Create auth handler instance in `routes.go`.
- Mount public auth pair routes before bearer-protected route group.
- Wrap protected `/api` group with `BearerGate`.
- In `main.go`, require non-empty master password before starting server.
- Initialize token store path under `~/.config/gitty/auth-tokens.json`.
- Create pairing session and print QR payload at startup.

- [ ] **Step 4: Run tests to verify pass**

Run:
- `cd server && go test ./internal/api -run NewRouter -v`
- `cd server && go test ./internal/config -run MasterPassword -v`

Expected: PASS

- [ ] **Step 5: Commit route/bootstrap unit**

```bash
git add server/internal/api/routes.go server/internal/api/routes_test.go server/cmd/gittyd/main.go server/internal/config/config.go server/internal/config/config_test.go
git commit -m "feat(auth): enforce bearer-protected api routing with startup qr bootstrap"
```

---

### Task 6: OpenAPI and Backend Documentation

**Files:**
- Modify: `server/internal/api/openapi.yml`
- Modify: `server/README.md`

- [ ] **Step 1: Add auth API and security scheme to OpenAPI**

Add:
- `components.securitySchemes.BearerAuth`
- `/api/auth/pair/exchange`, `/api/auth/devices`, `/api/auth/devices/{deviceId}`
- Security requirements on protected endpoints

- [ ] **Step 2: Update backend README**

Document:
- Startup QR pairing behavior
- Bearer auth header requirement
- Device revoke endpoint
- Public vs protected endpoint policy

- [ ] **Step 3: Validate OpenAPI syntax quickly**

Run: `cd server && go test ./internal/api -run OpenAPI -v` (or existing openapi-related tests)  
Expected: PASS or no matching tests; if no tests, note manual schema review completed.

- [ ] **Step 4: Commit API docs update**

```bash
git add server/internal/api/openapi.yml server/README.md
git commit -m "docs(api): document qr pairing and bearer auth contract"
```

---

### Task 7: End-to-End Verification and Regression Pass

**Files:**
- Test only; no required file edits unless failures found

- [ ] **Step 1: Run focused auth test suites**

Run:
- `cd server && go test ./internal/auth ./internal/api ./internal/api/handlers ./internal/config -v`

Expected: PASS

- [ ] **Step 2: Run full backend tests**

Run:
- `cd server && go test ./...`

Expected: PASS across all backend packages

- [ ] **Step 3: Smoke-check startup behavior manually**

Run:
- `cd server && GITTY_MASTER_PASSWORD=secret go run ./cmd/gittyd`

Expected:
- Startup logs include pairing session output and QR payload
- `/health` reachable without token
- `/api/repos` returns `401` without token
- Pair exchange returns bearer token
- Same token authorizes `/api/repos`

- [ ] **Step 4: Commit any final fixes**

```bash
git add -A
git commit -m "test(auth): finalize qr bearer auth verification fixes"
```

---

## Final Validation Checklist

- [ ] No protected `/api/*` endpoint bypasses bearer middleware.
- [ ] Pair exchange requires both valid session and correct `masterPassword`.
- [ ] Raw tokens are never written to disk or logs.
- [ ] Tokens remain valid across restart and can be revoked per device.
- [ ] OpenAPI + README match implemented behavior.

---

## Suggested Commit Sequence

1. `feat(auth): add ephemeral pairing session manager`
2. `feat(auth): add persistent per-device token store`
3. `feat(auth): add bearer middleware with context device binding`
4. `feat(api): add auth pairing and device management handlers`
5. `feat(auth): enforce bearer-protected api routing with startup qr bootstrap`
6. `docs(api): document qr pairing and bearer auth contract`
7. `test(auth): finalize qr bearer auth verification fixes`
