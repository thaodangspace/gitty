package api

import (
    "encoding/json"
    "net/http"
    "net/http/httptest"
    "testing"

    "gitweb/server/internal/models"
)

func TestNewRouterListRepositories(t *testing.T) {
    tempDir := t.TempDir()
    r := NewRouter(tempDir)

    req := httptest.NewRequest(http.MethodGet, "/api/repos", nil)
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

