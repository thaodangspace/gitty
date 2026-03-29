package auth

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"sync"
	"time"
)

// ErrPairSessionUnavailable is returned when a pairing session is missing, expired, or already used.
var ErrPairSessionUnavailable = errors.New("pairing session unavailable")

// PairSession represents an ephemeral QR pairing session.
type PairSession struct {
	SessionID string
	CreatedAt time.Time
	ExpiresAt time.Time
	UsedAt    *time.Time
}

// PairingManager manages ephemeral pairing sessions.
type PairingManager struct {
	mu       sync.Mutex
	sessions map[string]*PairSession
	ttl      time.Duration
}

// NewPairingManager creates a new PairingManager with the given session TTL.
func NewPairingManager(ttl time.Duration) *PairingManager {
	return &PairingManager{
		sessions: make(map[string]*PairSession),
		ttl:      ttl,
	}
}

// CreateSession generates a new ephemeral pairing session with a crypto-random ID.
func (m *PairingManager) CreateSession() (*PairSession, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return nil, err
	}

	now := time.Now()
	sess := &PairSession{
		SessionID: hex.EncodeToString(b),
		CreatedAt: now,
		ExpiresAt: now.Add(m.ttl),
	}

	m.mu.Lock()
	m.sessions[sess.SessionID] = sess
	m.mu.Unlock()

	return sess, nil
}

// Validate returns the session if it exists, is not expired, and has not been used.
// Returns ErrPairSessionUnavailable otherwise.
// The full check is performed under the mutex to prevent data races with MarkUsed.
func (m *PairingManager) Validate(sessionID string) (*PairSession, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	sess, ok := m.sessions[sessionID]
	if !ok {
		return nil, ErrPairSessionUnavailable
	}
	if time.Now().After(sess.ExpiresAt) {
		return nil, ErrPairSessionUnavailable
	}
	if sess.UsedAt != nil {
		return nil, ErrPairSessionUnavailable
	}

	return sess, nil
}

// ValidateAndConsume atomically validates the session and marks it as used in a
// single locked operation, preventing TOCTOU races where two concurrent callers
// could both pass Validate before either calls MarkUsed.
// Returns ErrPairSessionUnavailable if the session is missing, expired, or already used.
func (m *PairingManager) ValidateAndConsume(sessionID string) (*PairSession, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	sess, ok := m.sessions[sessionID]
	if !ok {
		return nil, ErrPairSessionUnavailable
	}
	if time.Now().After(sess.ExpiresAt) {
		return nil, ErrPairSessionUnavailable
	}
	if sess.UsedAt != nil {
		return nil, ErrPairSessionUnavailable
	}

	now := time.Now()
	sess.UsedAt = &now
	return sess, nil
}

// MarkUsed marks the session as consumed so it cannot be validated again.
// Returns ErrPairSessionUnavailable if the session does not exist.
func (m *PairingManager) MarkUsed(sessionID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	sess, ok := m.sessions[sessionID]
	if !ok {
		return ErrPairSessionUnavailable
	}

	now := time.Now()
	sess.UsedAt = &now
	return nil
}
