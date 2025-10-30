package auth

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestPasswordGateAllowsWhenPasswordMatchesHeader(t *testing.T) {
	handlerCalled := false
	gate := PasswordGate("secret")
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("X-Gitty-Master", "secret")

	rr := httptest.NewRecorder()
	gate(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		handlerCalled = true
		w.WriteHeader(http.StatusTeapot)
	})).ServeHTTP(rr, req)

	if !handlerCalled {
		t.Fatalf("expected handler to be called")
	}

	if rr.Code != http.StatusTeapot {
		t.Fatalf("expected status %d, got %d", http.StatusTeapot, rr.Code)
	}
}

func TestPasswordGateAllowsWhenPasswordMatchesBasicAuth(t *testing.T) {
	handlerCalled := false
	gate := PasswordGate("secret")
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.SetBasicAuth("", "secret")

	rr := httptest.NewRecorder()
	gate(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		handlerCalled = true
		w.WriteHeader(http.StatusAccepted)
	})).ServeHTTP(rr, req)

	if !handlerCalled {
		t.Fatalf("expected handler to be called")
	}

	if rr.Code != http.StatusAccepted {
		t.Fatalf("expected status %d, got %d", http.StatusAccepted, rr.Code)
	}
}

func TestPasswordGateRejectsWhenMissing(t *testing.T) {
	gate := PasswordGate("secret")
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rr := httptest.NewRecorder()

	gate(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatalf("handler should not be called")
	})).ServeHTTP(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("expected status %d, got %d", http.StatusUnauthorized, rr.Code)
	}

	if got := rr.Body.String(); got == "" {
		t.Fatalf("expected error message in body")
	}
}

func TestPasswordGatePassThroughWhenNotConfigured(t *testing.T) {
	handlerCalled := false
	gate := PasswordGate("")
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rr := httptest.NewRecorder()

	gate(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		handlerCalled = true
	})).ServeHTTP(rr, req)

	if !handlerCalled {
		t.Fatalf("expected handler to be called when password is not configured")
	}
}
