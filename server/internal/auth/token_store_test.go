package auth

import (
	"path/filepath"
	"testing"
)

func TestTokenStore_IssueValidateRevoke(t *testing.T) {
	dir := t.TempDir()
	store, err := NewTokenStore(filepath.Join(dir, "auth-tokens.json"))
	if err != nil {
		t.Fatalf("new token store: %v", err)
	}

	token, device, err := store.IssueToken("iPhone")
	if err != nil {
		t.Fatalf("issue token: %v", err)
	}
	if token == "" || device.DeviceID == "" {
		t.Fatalf("expected issued token and device id")
	}

	got, ok := store.Validate(token)
	if !ok || got.DeviceID != device.DeviceID {
		t.Fatalf("validate failed")
	}

	if err := store.Revoke(device.DeviceID); err != nil {
		t.Fatalf("revoke failed: %v", err)
	}

	if _, ok := store.Validate(token); ok {
		t.Fatalf("expected revoked token to fail validation")
	}
}

func TestTokenStore_PersistsAcrossRestart(t *testing.T) {
	path := filepath.Join(t.TempDir(), "auth-tokens.json")

	store1, err := NewTokenStore(path)
	if err != nil {
		t.Fatalf("new token store: %v", err)
	}
	token, device, err := store1.IssueToken("Pixel")
	if err != nil {
		t.Fatalf("issue token: %v", err)
	}

	store2, err := NewTokenStore(path)
	if err != nil {
		t.Fatalf("new token store (restart): %v", err)
	}
	got, ok := store2.Validate(token)
	if !ok || got.DeviceID != device.DeviceID {
		t.Fatalf("expected token to survive restart")
	}
}
