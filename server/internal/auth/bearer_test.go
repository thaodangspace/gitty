package auth

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

// fakeValidator implements TokenValidator for testing.
type fakeValidator struct {
	allow  bool
	record DeviceTokenRecord
}

func (f fakeValidator) Validate(rawToken string) (DeviceTokenRecord, bool) {
	if f.allow {
		return f.record, true
	}
	return DeviceTokenRecord{}, false
}

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
	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rr.Code)
	}
}

func TestBearerGate_RejectsMalformedHeader(t *testing.T) {
	tests := []struct {
		name      string
		header    string
		validator TokenValidator
	}{
		{"no scheme", "justtokennoscheme", fakeValidator{allow: true}},
		{"wrong scheme", "Basic dXNlcjpwYXNz", fakeValidator{allow: true}},
		{"bearer no token", "Bearer ", fakeValidator{allow: true}},
		{"bearer with spaces only", "Bearer    ", fakeValidator{allow: true}},
		{"token with internal spaces", "Bearer tok en", fakeValidator{allow: false}},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			gate := BearerGate(tc.validator)
			rr := httptest.NewRecorder()
			req := httptest.NewRequest(http.MethodGet, "/api/repos", nil)
			req.Header.Set("Authorization", tc.header)

			gate(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
				t.Fatal("handler should not run")
			})).ServeHTTP(rr, req)

			if rr.Code != http.StatusUnauthorized {
				t.Fatalf("expected 401, got %d", rr.Code)
			}
		})
	}
}

func TestBearerGate_RejectsInvalidToken(t *testing.T) {
	gate := BearerGate(fakeValidator{allow: false})
	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/repos", nil)
	req.Header.Set("Authorization", "Bearer badtoken")

	gate(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		t.Fatal("handler should not run")
	})).ServeHTTP(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", rr.Code)
	}
}

func TestBearerGate_StoresDeviceInContext(t *testing.T) {
	expected := DeviceTokenRecord{
		DeviceID:   "device-abc",
		DeviceName: "my-phone",
	}
	gate := BearerGate(fakeValidator{allow: true, record: expected})
	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/repos", nil)
	req.Header.Set("Authorization", "Bearer valid")

	var got DeviceTokenRecord
	var contextOk bool
	gate(http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
		got, contextOk = DeviceFromContext(r.Context())
	})).ServeHTTP(rr, req)

	if !contextOk {
		t.Fatal("expected DeviceTokenRecord in context")
	}
	if got.DeviceID != expected.DeviceID || got.DeviceName != expected.DeviceName {
		t.Fatalf("context record mismatch: got %+v, want %+v", got, expected)
	}
}

func TestDeviceFromContext_MissingReturnsNotOk(t *testing.T) {
	_, ok := DeviceFromContext(context.Background())
	if ok {
		t.Fatal("expected false for empty context")
	}
}

func TestBearerGate_ResponseBody_IsJSON(t *testing.T) {
	gate := BearerGate(fakeValidator{})
	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/repos", nil)

	gate(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {})).ServeHTTP(rr, req)

	if ct := rr.Header().Get("Content-Type"); ct != "application/json" {
		t.Fatalf("expected Content-Type application/json, got %q", ct)
	}
	body := rr.Body.String()
	if body != `{"error":"unauthorized"}`+"\n" {
		t.Fatalf("unexpected body: %q", body)
	}
}
