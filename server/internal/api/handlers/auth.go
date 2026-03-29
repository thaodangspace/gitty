package handlers

import (
	"crypto/subtle"
	"encoding/json"
	"net/http"

	"gitweb/server/internal/auth"

	"github.com/go-chi/chi/v5"
)

// AuthHandler handles authentication-related HTTP endpoints.
type AuthHandler struct {
	pairingManager *auth.PairingManager
	tokenStore     *auth.TokenStore
	masterPassword string
}

// NewAuthHandler creates a new AuthHandler with the given dependencies.
func NewAuthHandler(pm *auth.PairingManager, ts *auth.TokenStore, masterPassword string) *AuthHandler {
	return &AuthHandler{
		pairingManager: pm,
		tokenStore:     ts,
		masterPassword: masterPassword,
	}
}

// PairExchangeRequest represents the JSON body for the pair exchange endpoint.
type PairExchangeRequest struct {
	SessionID      string `json:"sessionId"`
	MasterPassword string `json:"masterPassword"`
	DeviceName     string `json:"deviceName"`
}

// PairExchangeResponse represents the JSON response for the pair exchange endpoint.
type PairExchangeResponse struct {
	Token      string `json:"token"`
	DeviceID   string `json:"deviceId"`
}

// PairExchange handles POST /api/auth/pair/exchange.
// It validates the session and master password, then issues a new device token.
func (h *AuthHandler) PairExchange(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req PairExchangeRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid JSON", http.StatusBadRequest)
		return
	}

	// Validate all fields are non-empty
	if req.SessionID == "" || req.MasterPassword == "" || req.DeviceName == "" {
		http.Error(w, "sessionId, masterPassword, and deviceName are required", http.StatusBadRequest)
		return
	}

	// Constant-time comparison of master password
	if subtle.ConstantTimeCompare([]byte(req.MasterPassword), []byte(h.masterPassword)) != 1 {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	// Validate and consume the pairing session
	if _, err := h.pairingManager.ValidateAndConsume(req.SessionID); err != nil {
		if err == auth.ErrPairSessionUnavailable {
			http.Error(w, "session invalid or expired", http.StatusBadRequest)
			return
		}
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	// Issue a new token for the device
	rawToken, rec, err := h.tokenStore.IssueToken(req.DeviceName)
	if err != nil {
		http.Error(w, "failed to issue token", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(PairExchangeResponse{
		Token:    rawToken,
		DeviceID: rec.DeviceID,
	}); err != nil {
		http.Error(w, "failed to encode response", http.StatusInternalServerError)
		return
	}
}

// ListDevices handles GET /api/auth/devices.
// It returns all device records without including token hashes.
func (h *AuthHandler) ListDevices(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	records := h.tokenStore.List()

	// Zero out token hashes before returning
	for i := range records {
		records[i].TokenHash = ""
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(records); err != nil {
		http.Error(w, "failed to encode response", http.StatusInternalServerError)
		return
	}
}

// RevokeDevice handles DELETE /api/auth/devices/{deviceId}.
// It revokes the device token and returns 204 No Content on success.
func (h *AuthHandler) RevokeDevice(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodDelete {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	deviceID := chi.URLParam(r, "deviceId")
	if deviceID == "" {
		http.Error(w, "deviceId required", http.StatusBadRequest)
		return
	}

	if err := h.tokenStore.Revoke(deviceID); err != nil {
		// Check if the error indicates the device was not found
		if err.Error() == `device "`+deviceID+`" not found` {
			http.Error(w, "device not found", http.StatusNotFound)
			return
		}
		http.Error(w, "failed to revoke device", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
