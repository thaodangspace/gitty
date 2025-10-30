package auth

import (
	"crypto/subtle"
	"net/http"
	"strings"
)

const masterHeader = "X-Gitty-Master"

// PasswordGate returns a middleware that checks the provided credential against the master password.
// The credential can be supplied either via the custom X-Gitty-Master header or via HTTP basic auth.
// If the master password is empty, the middleware allows all requests to proceed.
func PasswordGate(masterPassword string) func(http.Handler) http.Handler {
	trimmedPassword := strings.TrimSpace(masterPassword)

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if trimmedPassword == "" {
				next.ServeHTTP(w, r)
				return
			}

			if credential := extractCredential(r); credential != "" && secureCompare(credential, trimmedPassword) {
				next.ServeHTTP(w, r)
				return
			}

			w.Header().Set("WWW-Authenticate", `Basic realm="gitty", charset="UTF-8"`)
			http.Error(w, "missing or invalid master password", http.StatusUnauthorized)
		})
	}
}

func extractCredential(r *http.Request) string {
	if header := strings.TrimSpace(r.Header.Get(masterHeader)); header != "" {
		return header
	}

	if username, password, ok := r.BasicAuth(); ok {
		if username == "" {
			return password
		}
		// allow clients to send either username or password as credential, prefer password when provided.
		if password != "" {
			return password
		}
		return username
	}

	return ""
}

func secureCompare(a, b string) bool {
	if len(a) != len(b) {
		return false
	}

	return subtle.ConstantTimeCompare([]byte(a), []byte(b)) == 1
}
