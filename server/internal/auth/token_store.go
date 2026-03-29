package auth

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// DeviceTokenRecord holds the persisted record for a single device token.
// The raw token is never stored; only its SHA-256 hash is persisted.
type DeviceTokenRecord struct {
	DeviceID   string     `json:"deviceId"`
	DeviceName string     `json:"deviceName"`
	TokenHash  string     `json:"tokenHash"`
	CreatedAt  time.Time  `json:"createdAt"`
	LastUsedAt *time.Time `json:"lastUsedAt,omitempty"`
	RevokedAt  *time.Time `json:"revokedAt,omitempty"`
}

// TokenStore manages per-device bearer tokens with file-backed persistence.
// Raw tokens are never stored; only SHA-256 hashes are written to disk.
type TokenStore struct {
	mu      sync.Mutex
	path    string
	records []DeviceTokenRecord
}

// NewTokenStore opens (or creates) the token store at path, loading any
// existing records from disk.
func NewTokenStore(path string) (*TokenStore, error) {
	s := &TokenStore{path: path}
	if err := s.load(); err != nil {
		return nil, fmt.Errorf("load token store: %w", err)
	}
	return s, nil
}

// IssueToken generates a new crypto-random bearer token for deviceName,
// stores its SHA-256 hash, persists to disk, and returns the raw token.
func (s *TokenStore) IssueToken(deviceName string) (rawToken string, rec DeviceTokenRecord, err error) {
	idBytes := make([]byte, 16)
	if _, err = rand.Read(idBytes); err != nil {
		return "", DeviceTokenRecord{}, fmt.Errorf("generate device id: %w", err)
	}

	tokenBytes := make([]byte, 32)
	if _, err = rand.Read(tokenBytes); err != nil {
		return "", DeviceTokenRecord{}, fmt.Errorf("generate token: %w", err)
	}

	rawToken = hex.EncodeToString(tokenBytes)
	hash := hashToken(rawToken)

	rec = DeviceTokenRecord{
		DeviceID:   hex.EncodeToString(idBytes),
		DeviceName: deviceName,
		TokenHash:  hash,
		CreatedAt:  time.Now().UTC(),
	}

	s.mu.Lock()
	s.records = append(s.records, rec)
	if saveErr := s.save(); saveErr != nil {
		s.records = s.records[:len(s.records)-1] // roll back
		s.mu.Unlock()
		return "", DeviceTokenRecord{}, fmt.Errorf("persist token: %w", saveErr)
	}
	s.mu.Unlock()

	return rawToken, rec, nil
}

// Validate checks rawToken against stored hashes. If a match is found and the
// token is not revoked, LastUsedAt is updated and the record is returned.
func (s *TokenStore) Validate(rawToken string) (DeviceTokenRecord, bool) {
	hash := hashToken(rawToken)

	s.mu.Lock()
	defer s.mu.Unlock()

	for i, r := range s.records {
		if r.TokenHash == hash && r.RevokedAt == nil {
			now := time.Now().UTC()
			s.records[i].LastUsedAt = &now
			// LastUsedAt is audit-only; persist failure does not reject a valid token
			_ = s.save()
			return s.records[i], true
		}
	}
	return DeviceTokenRecord{}, false
}

// List returns all token records, including revoked ones.
func (s *TokenStore) List() []DeviceTokenRecord {
	s.mu.Lock()
	defer s.mu.Unlock()

	out := make([]DeviceTokenRecord, len(s.records))
	copy(out, s.records)
	return out
}

// Revoke marks the token for deviceID as revoked and persists the change.
func (s *TokenStore) Revoke(deviceID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	for i, r := range s.records {
		if r.DeviceID == deviceID {
			now := time.Now().UTC()
			s.records[i].RevokedAt = &now
			return s.save()
		}
	}
	return fmt.Errorf("device %q not found", deviceID)
}

// IssueTokenWithID is a test helper that creates a token record with a specific device ID.
// Only for use in tests — do not use in production code.
func (s *TokenStore) IssueTokenWithID(deviceID, deviceName, rawToken string) (DeviceTokenRecord, error) {
	hash := hashToken(rawToken)

	rec := DeviceTokenRecord{
		DeviceID:   deviceID,
		DeviceName: deviceName,
		TokenHash:  hash,
		CreatedAt:  time.Now().UTC(),
	}

	s.mu.Lock()
	s.records = append(s.records, rec)
	if err := s.save(); err != nil {
		s.records = s.records[:len(s.records)-1] // roll back
		s.mu.Unlock()
		return DeviceTokenRecord{}, err
	}
	s.mu.Unlock()

	return rec, nil
}

// hashToken returns the hex-encoded SHA-256 digest of rawToken.
func hashToken(rawToken string) string {
	sum := sha256.Sum256([]byte(rawToken))
	return hex.EncodeToString(sum[:])
}

// load reads and parses the JSON store file. A missing file is treated as an
// empty store (not an error).
func (s *TokenStore) load() error {
	data, err := os.ReadFile(s.path)
	if errors.Is(err, os.ErrNotExist) {
		s.records = nil
		return nil
	}
	if err != nil {
		return err
	}
	return json.Unmarshal(data, &s.records)
}

// save atomically writes the current records to disk using a temp file + rename.
// Must be called with s.mu held.
func (s *TokenStore) save() error {
	data, err := json.MarshalIndent(s.records, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal records: %w", err)
	}

	dir := filepath.Dir(s.path)
	tmp, err := os.CreateTemp(dir, ".auth-tokens-*.json")
	if err != nil {
		return fmt.Errorf("create temp file: %w", err)
	}
	tmpName := tmp.Name()

	// Ensure temp file is cleaned up on failure.
	defer func() {
		if err != nil {
			_ = os.Remove(tmpName)
		}
	}()

	if err = tmp.Chmod(0600); err != nil {
		_ = tmp.Close()
		return fmt.Errorf("chmod temp file: %w", err)
	}
	if _, err = tmp.Write(data); err != nil {
		_ = tmp.Close()
		return fmt.Errorf("write temp file: %w", err)
	}
	if err = tmp.Close(); err != nil {
		return fmt.Errorf("close temp file: %w", err)
	}

	if err = os.Rename(tmpName, s.path); err != nil {
		return fmt.Errorf("rename temp file: %w", err)
	}
	return nil
}
