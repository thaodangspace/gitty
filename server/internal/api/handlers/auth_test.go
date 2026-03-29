package handlers

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"gitweb/server/internal/auth"

	"github.com/go-chi/chi/v5"
)

func newTestAuthHandler(t *testing.T) *AuthHandler {
	t.Helper()

	// Create a real PairingManager with an active session "s1"
	pairingManager := auth.NewPairingManager(5 * time.Minute)

	// Create session with known ID "s1" using the test helper
	_, err := pairingManager.CreateSessionWithID("s1")
	if err != nil {
		t.Fatalf("failed to create session: %v", err)
	}

	// Create a temp directory for the token store
	tempDir := t.TempDir()
	tokenStore, err := auth.NewTokenStore(tempDir + "/tokens.json")
	if err != nil {
		t.Fatalf("failed to create token store: %v", err)
	}

	// Pre-issue a device with ID "dev_1" for the RevokeDevice test
	_, err = tokenStore.IssueTokenWithID("dev_1", "test-device", "test-token-secret")
	if err != nil {
		t.Fatalf("failed to issue token: %v", err)
	}

	// Create the auth handler
	handler := &AuthHandler{
		pairingManager: pairingManager,
		tokenStore:     tokenStore,
		masterPassword: "secret",
	}

	return handler
}

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

	// Create request with chi router context to properly extract URL params
	r := chi.NewRouter()
	r.Delete("/api/auth/devices/{deviceId}", h.RevokeDevice)

	req := httptest.NewRequest(http.MethodDelete, "/api/auth/devices/dev_1", nil)
	rr := httptest.NewRecorder()

	r.ServeHTTP(rr, req)

	if rr.Code != http.StatusNoContent {
		t.Fatalf("expected 204, got %d", rr.Code)
	}
}

func TestAuthHandler_PairExchange_MissingFields(t *testing.T) {
	h := newTestAuthHandler(t)

	// Missing sessionId
	body := strings.NewReader(`{"masterPassword":"secret","deviceName":"iPhone"}`)
	req := httptest.NewRequest(http.MethodPost, "/api/auth/pair/exchange", body)
	rr := httptest.NewRecorder()

	h.PairExchange(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for missing sessionId, got %d", rr.Code)
	}

	// Missing masterPassword
	body = strings.NewReader(`{"sessionId":"s1","deviceName":"iPhone"}`)
	req = httptest.NewRequest(http.MethodPost, "/api/auth/pair/exchange", body)
	rr = httptest.NewRecorder()

	h.PairExchange(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for missing masterPassword, got %d", rr.Code)
	}

	// Missing deviceName
	body = strings.NewReader(`{"sessionId":"s1","masterPassword":"secret"}`)
	req = httptest.NewRequest(http.MethodPost, "/api/auth/pair/exchange", body)
	rr = httptest.NewRecorder()

	h.PairExchange(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for missing deviceName, got %d", rr.Code)
	}
}

func TestAuthHandler_PairExchange_WrongPassword(t *testing.T) {
	h := newTestAuthHandler(t)

	body := strings.NewReader(`{"sessionId":"s1","masterPassword":"wrong","deviceName":"iPhone"}`)
	req := httptest.NewRequest(http.MethodPost, "/api/auth/pair/exchange", body)
	rr := httptest.NewRecorder()

	h.PairExchange(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 for wrong password, got %d", rr.Code)
	}
}

func TestAuthHandler_PairExchange_InvalidSession(t *testing.T) {
	h := newTestAuthHandler(t)

	body := strings.NewReader(`{"sessionId":"invalid","masterPassword":"secret","deviceName":"iPhone"}`)
	req := httptest.NewRequest(http.MethodPost, "/api/auth/pair/exchange", body)
	rr := httptest.NewRecorder()

	h.PairExchange(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for invalid session, got %d", rr.Code)
	}
}

func TestAuthHandler_PairExchange_SessionAlreadyUsed(t *testing.T) {
	h := newTestAuthHandler(t)

	// First request should succeed
	body := strings.NewReader(`{"sessionId":"s1","masterPassword":"secret","deviceName":"iPhone"}`)
	req := httptest.NewRequest(http.MethodPost, "/api/auth/pair/exchange", body)
	rr := httptest.NewRecorder()

	h.PairExchange(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("first request expected 200, got %d", rr.Code)
	}

	// Second request with same session should fail
	body = strings.NewReader(`{"sessionId":"s1","masterPassword":"secret","deviceName":"Android"}`)
	req = httptest.NewRequest(http.MethodPost, "/api/auth/pair/exchange", body)
	rr = httptest.NewRecorder()

	h.PairExchange(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("second request expected 400 (session used), got %d", rr.Code)
	}
}

func TestAuthHandler_RevokeDevice_NotFound(t *testing.T) {
	h := newTestAuthHandler(t)

	r := chi.NewRouter()
	r.Delete("/api/auth/devices/{deviceId}", h.RevokeDevice)

	req := httptest.NewRequest(http.MethodDelete, "/api/auth/devices/nonexistent", nil)
	rr := httptest.NewRecorder()

	r.ServeHTTP(rr, req)

	if rr.Code != http.StatusNotFound {
		t.Fatalf("expected 404 for nonexistent device, got %d", rr.Code)
	}
}

func TestAuthHandler_ListDevices(t *testing.T) {
	h := newTestAuthHandler(t)

	req := httptest.NewRequest(http.MethodGet, "/api/auth/devices", nil)
	rr := httptest.NewRecorder()

	h.ListDevices(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rr.Code)
	}

	// Verify response structure and that tokenHash is omitted
	var devices []auth.DeviceTokenRecord
	if err := json.NewDecoder(rr.Body).Decode(&devices); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if len(devices) != 1 {
		t.Fatalf("expected 1 device, got %d", len(devices))
	}

	if devices[0].DeviceID != "dev_1" {
		t.Fatalf("expected device ID dev_1, got %s", devices[0].DeviceID)
	}

	// Verify tokenHash is empty
	if devices[0].TokenHash != "" {
		t.Fatalf("expected empty tokenHash, got %s", devices[0].TokenHash)
	}
}
