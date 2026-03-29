package api

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"gitweb/server/internal/auth"
	"gitweb/server/internal/config"
	"gitweb/server/internal/models"
	"gitweb/server/internal/registry"
)

func TestNewRouterListRepositories(t *testing.T) {
	tempDir := t.TempDir()
	cfg := &config.Config{}
	masterPassword := "test-secret"
	cfg.MasterPassword = &masterPassword
	pm := auth.NewPairingManager(auth.DefaultPairSessionTTL)
	ts, _ := auth.NewTokenStore(tempDir + "/tokens.json")
	reg, _ := registry.New(tempDir + "/registry.json")

	// Issue a token for testing
	rawToken, _, err := ts.IssueToken("test-device")
	if err != nil {
		t.Fatalf("failed to issue token: %v", err)
	}

	r := NewRouter(context.Background(), tempDir, cfg, reg, pm, ts)

	req := httptest.NewRequest(http.MethodGet, "/api/repos", nil)
	req.Header.Set("Authorization", "Bearer "+rawToken)
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, rr.Code)
	}

	var repos []models.Repository
	if err := json.Unmarshal(rr.Body.Bytes(), &repos); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if len(repos) != 0 {
		t.Fatalf("expected empty repository list, got %d", len(repos))
	}
}

// TestHealthEndpointOpenWithoutBearer verifies /health is publicly accessible without bearer token.
func TestHealthEndpointOpenWithoutBearer(t *testing.T) {
	tempDir := t.TempDir()
	cfg := &config.Config{}
	masterPassword := "test-secret"
	cfg.MasterPassword = &masterPassword
	pm := auth.NewPairingManager(auth.DefaultPairSessionTTL)
	ts, _ := auth.NewTokenStore(tempDir + "/tokens.json")
	reg, _ := registry.New(tempDir + "/registry.json")

	r := NewRouter(context.Background(), tempDir, cfg, reg, pm, ts)

	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected /health to return 200 without bearer, got %d", rr.Code)
	}
}

// TestAuthPairExchangeOpenWithoutBearer verifies /api/auth/pair/exchange is publicly accessible.
func TestAuthPairExchangeOpenWithoutBearer(t *testing.T) {
	tempDir := t.TempDir()
	cfg := &config.Config{}
	pm := auth.NewPairingManager(auth.DefaultPairSessionTTL)
	ts, _ := auth.NewTokenStore(tempDir + "/tokens.json")
	masterPassword := "test-secret"
	cfg.MasterPassword = &masterPassword
	reg, _ := registry.New(tempDir + "/registry.json")

	r := NewRouter(context.Background(), tempDir, cfg, reg, pm, ts)

	req := httptest.NewRequest(http.MethodPost, "/api/auth/pair/exchange", nil)
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)

	// Should not return 401 (bearer required) - may return 400 for invalid JSON
	if rr.Code == http.StatusUnauthorized {
		t.Fatalf("expected /api/auth/pair/exchange to be open without bearer, got 401")
	}
}

// TestApiReposRequiresBearer verifies /api/repos returns 401 without bearer token.
func TestApiReposRequiresBearer(t *testing.T) {
	tempDir := t.TempDir()
	cfg := &config.Config{}
	pm := auth.NewPairingManager(auth.DefaultPairSessionTTL)
	ts, _ := auth.NewTokenStore(tempDir + "/tokens.json")
	masterPassword := "test-secret"
	cfg.MasterPassword = &masterPassword
	reg, _ := registry.New(tempDir + "/registry.json")

	r := NewRouter(context.Background(), tempDir, cfg, reg, pm, ts)

	req := httptest.NewRequest(http.MethodGet, "/api/repos", nil)
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("expected /api/repos to return 401 without bearer, got %d", rr.Code)
	}
}
