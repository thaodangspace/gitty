package handlers

import (
	"crypto/subtle"
	"encoding/json"
	"fmt"
	"log"
	"net"
	"net/http"
	"time"

	"gitweb/server/internal/auth"

	"github.com/go-chi/chi/v5"
)

// AuthHandler handles authentication-related HTTP endpoints.
type AuthHandler struct {
	pairingManager  *auth.PairingManager
	tokenStore      *auth.TokenStore
	masterPassword  string
	currentSessionID string // Session ID created at startup for QR code
}

// NewAuthHandler creates a new AuthHandler with the given dependencies.
func NewAuthHandler(pm *auth.PairingManager, ts *auth.TokenStore, masterPassword string) *AuthHandler {
	return &AuthHandler{
		pairingManager: pm,
		tokenStore:     ts,
		masterPassword: masterPassword,
		currentSessionID: "", // Will be set via SetCurrentSessionID
	}
}

// SetCurrentSessionID sets the current pairing session ID (called after startup session creation)
func (h *AuthHandler) SetCurrentSessionID(sessionID string) {
	h.currentSessionID = sessionID
}

// NewAuthHandlerWithSession creates a new AuthHandler with an active session ID.
// This is used when the server creates a pairing session at startup.
func NewAuthHandlerWithSession(pm *auth.PairingManager, ts *auth.TokenStore, masterPassword string, sessionID string) *AuthHandler {
	return &AuthHandler{
		pairingManager:   pm,
		tokenStore:       ts,
		masterPassword:   masterPassword,
		currentSessionID: sessionID,
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

// PairSessionResponse represents the JSON response for the pair session endpoint.
type PairSessionResponse struct {
	SessionID string `json:"sessionId"`
	ExpiresAt string `json:"expiresAt"`
}

// GetPairSession handles GET /api/auth/pair/session.
// It returns the current pairing session information for QR code scanning.
func (h *AuthHandler) GetPairSession(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	if h.currentSessionID == "" {
		http.Error(w, "no active pairing session", http.StatusServiceUnavailable)
		return
	}

	// Validate the session is still active (not expired or used)
	sess, err := h.pairingManager.Validate(h.currentSessionID)
	if err != nil {
		http.Error(w, "session expired or invalid", http.StatusGone)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(PairSessionResponse{
		SessionID: sess.SessionID,
		ExpiresAt: sess.ExpiresAt.Format(time.RFC3339),
	}); err != nil {
		http.Error(w, "failed to encode response", http.StatusInternalServerError)
		return
	}
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
	log.Printf("[DEBUG] received password: %q, expected: %q", req.MasterPassword, h.masterPassword)
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

// LocalPairRequest represents the JSON body for the local pairing endpoint.
type LocalPairRequest struct {
	MasterPassword string `json:"masterPassword"`
}

// LocalPair handles POST /api/auth/local/pair.
// It issues a device token for localhost connections after validating the master password.
// This is a simplified flow for the web frontend that runs on the same host.
func (h *AuthHandler) LocalPair(w http.ResponseWriter, r *http.Request) {
	// Only allow from localhost
	if !isLocalhost(r) {
		http.Error(w, "forbidden: localhost only", http.StatusForbidden)
		return
	}

	var req LocalPairRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid JSON", http.StatusBadRequest)
		return
	}

	if req.MasterPassword == "" {
		http.Error(w, "masterPassword is required", http.StatusBadRequest)
		return
	}

	// Constant-time comparison of master password
	if subtle.ConstantTimeCompare([]byte(req.MasterPassword), []byte(h.masterPassword)) != 1 {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	deviceName := fmt.Sprintf("web-%d", time.Now().Unix())
	rawToken, rec, err := h.tokenStore.IssueToken(deviceName)
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

// isLocalhost checks whether the request originated from the loopback interface.
func isLocalhost(r *http.Request) bool {
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return false
	}
	return host == "127.0.0.1" || host == "::1" || host == "localhost"
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
