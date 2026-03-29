package auth

import (
	"errors"
	"testing"
	"time"
)

func TestPairingManager_CreateAndValidate(t *testing.T) {
	mgr := NewPairingManager(10 * time.Minute)

	sess, err := mgr.CreateSession()
	if err != nil {
		t.Fatalf("create session: %v", err)
	}

	got, err := mgr.Validate(sess.SessionID)
	if err != nil {
		t.Fatalf("validate session: %v", err)
	}
	if got.SessionID != sess.SessionID {
		t.Fatalf("session id mismatch")
	}
}

func TestPairingManager_SingleUse(t *testing.T) {
	mgr := NewPairingManager(10 * time.Minute)
	sess, _ := mgr.CreateSession()

	if err := mgr.MarkUsed(sess.SessionID); err != nil {
		t.Fatalf("mark used: %v", err)
	}

	if _, err := mgr.Validate(sess.SessionID); !errors.Is(err, ErrPairSessionUnavailable) {
		t.Fatalf("expected ErrPairSessionUnavailable, got %v", err)
	}
}

func TestPairingManager_Expired(t *testing.T) {
	mgr := NewPairingManager(1 * time.Millisecond)
	sess, err := mgr.CreateSession()
	if err != nil {
		t.Fatalf("create session: %v", err)
	}

	time.Sleep(5 * time.Millisecond)

	if _, err := mgr.Validate(sess.SessionID); !errors.Is(err, ErrPairSessionUnavailable) {
		t.Fatalf("expected ErrPairSessionUnavailable for expired session, got %v", err)
	}
}

func TestPairingManager_Missing(t *testing.T) {
	mgr := NewPairingManager(10 * time.Minute)

	if _, err := mgr.Validate("nonexistent-session-id"); !errors.Is(err, ErrPairSessionUnavailable) {
		t.Fatalf("expected ErrPairSessionUnavailable for missing session, got %v", err)
	}
}

func TestPairingManager_MarkUsed_Missing(t *testing.T) {
	mgr := NewPairingManager(10 * time.Minute)

	if err := mgr.MarkUsed("nonexistent-session-id"); !errors.Is(err, ErrPairSessionUnavailable) {
		t.Fatalf("expected ErrPairSessionUnavailable for missing session, got %v", err)
	}
}

func TestPairingManager_ValidateAndConsume(t *testing.T) {
	mgr := NewPairingManager(10 * time.Minute)
	sess, _ := mgr.CreateSession()

	// First call should succeed and return the session.
	got, err := mgr.ValidateAndConsume(sess.SessionID)
	if err != nil {
		t.Fatalf("ValidateAndConsume: %v", err)
	}
	if got.SessionID != sess.SessionID {
		t.Fatalf("session id mismatch")
	}

	// Second call must fail — session is now consumed.
	if _, err := mgr.ValidateAndConsume(sess.SessionID); !errors.Is(err, ErrPairSessionUnavailable) {
		t.Fatalf("expected ErrPairSessionUnavailable on second consume, got %v", err)
	}

	// Validate must also fail after consume.
	if _, err := mgr.Validate(sess.SessionID); !errors.Is(err, ErrPairSessionUnavailable) {
		t.Fatalf("expected ErrPairSessionUnavailable from Validate after consume, got %v", err)
	}
}

func TestPairingManager_ValidateAndConsume_Missing(t *testing.T) {
	mgr := NewPairingManager(10 * time.Minute)

	if _, err := mgr.ValidateAndConsume("nonexistent-session-id"); !errors.Is(err, ErrPairSessionUnavailable) {
		t.Fatalf("expected ErrPairSessionUnavailable for missing session, got %v", err)
	}
}

func TestPairingManager_UniqueSessionIDs(t *testing.T) {
	mgr := NewPairingManager(10 * time.Minute)

	sess1, err := mgr.CreateSession()
	if err != nil {
		t.Fatalf("create session 1: %v", err)
	}
	sess2, err := mgr.CreateSession()
	if err != nil {
		t.Fatalf("create session 2: %v", err)
	}

	if sess1.SessionID == sess2.SessionID {
		t.Fatalf("expected unique session IDs, got duplicates: %s", sess1.SessionID)
	}
}
