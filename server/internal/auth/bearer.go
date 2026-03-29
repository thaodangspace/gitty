package auth

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
)

// TokenValidator is the interface BearerGate uses to validate tokens.
// TokenStore implements this interface.
type TokenValidator interface {
	Validate(rawToken string) (DeviceTokenRecord, bool)
}

// contextKey is an unexported type for context keys in this package,
// preventing collisions with keys from other packages.
type contextKey int

const deviceContextKey contextKey = iota

// BearerGate returns middleware that enforces Bearer token authentication.
// Requests missing a valid "Authorization: Bearer <token>" header are rejected
// with 401 and a JSON error body. On success the DeviceTokenRecord is stored
// in the request context and can be retrieved with DeviceFromContext.
func BearerGate(validator TokenValidator) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			token, ok := extractBearerToken(r)
			if !ok {
				writeUnauthorized(w)
				return
			}

			record, valid := validator.Validate(token)
			if !valid {
				writeUnauthorized(w)
				return
			}

			ctx := context.WithValue(r.Context(), deviceContextKey, record)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// DeviceFromContext retrieves the DeviceTokenRecord stored by BearerGate
// from ctx. The second return value is false if no record is present.
func DeviceFromContext(ctx context.Context) (DeviceTokenRecord, bool) {
	rec, ok := ctx.Value(deviceContextKey).(DeviceTokenRecord)
	return rec, ok
}

// extractBearerToken parses the Authorization header and returns the raw token
// if the header is in the form "Bearer <token>". Returns ("", false) otherwise.
func extractBearerToken(r *http.Request) (string, bool) {
	header := r.Header.Get("Authorization")
	if header == "" {
		return "", false
	}

	prefix, token, found := strings.Cut(header, " ")
	if !found || !strings.EqualFold(prefix, "Bearer") || strings.TrimSpace(token) == "" {
		return "", false
	}

	return strings.TrimSpace(token), true
}

// writeUnauthorized writes a 401 JSON response.
func writeUnauthorized(w http.ResponseWriter) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusUnauthorized)
	_ = json.NewEncoder(w).Encode(map[string]string{"error": "unauthorized"})
}
