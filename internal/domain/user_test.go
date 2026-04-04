package domain

import (
	"testing"
)

func TestNewUser(t *testing.T) {
	t.Parallel()

	t.Run("generates_uuid", func(t *testing.T) {
		t.Parallel()
		u := NewUser("test@example.com")
		if u.UUID == "" {
			t.Error("NewUser().UUID is empty, want non-empty UUID")
		}
		if len(u.UUID) < 32 {
			t.Errorf("NewUser().UUID = %q, want valid UUID format", u.UUID)
		}
	})

	t.Run("sets_email", func(t *testing.T) {
		t.Parallel()
		u := NewUser("test@example.com")
		if u.Email != "test@example.com" {
			t.Errorf("NewUser().Email = %q, want %q", u.Email, "test@example.com")
		}
	})
}

func TestPseudoKeyParams(t *testing.T) {
	t.Parallel()

	t.Run("deterministic_nonce", func(t *testing.T) {
		t.Parallel()
		p1 := PseudoKeyParams("nonexistent@example.com")
		p2 := PseudoKeyParams("nonexistent@example.com")
		if p1.PwNonce != p2.PwNonce {
			t.Errorf("PseudoKeyParams nonce not deterministic: %q != %q", p1.PwNonce, p2.PwNonce)
		}
	})

	t.Run("different_emails_different_nonce", func(t *testing.T) {
		t.Parallel()
		p1 := PseudoKeyParams("user1@example.com")
		p2 := PseudoKeyParams("user2@example.com")
		if p1.PwNonce == p2.PwNonce {
			t.Errorf("different emails produced same nonce: %q", p1.PwNonce)
		}
	})

	t.Run("returns_valid_version", func(t *testing.T) {
		t.Parallel()
		p := PseudoKeyParams("test@example.com")
		if p.Version != "004" {
			t.Errorf("PseudoKeyParams.Version = %q, want %q", p.Version, "004")
		}
	})

	t.Run("sets_identifier", func(t *testing.T) {
		t.Parallel()
		p := PseudoKeyParams("test@example.com")
		if p.Identifier != "test@example.com" {
			t.Errorf("PseudoKeyParams.Identifier = %q, want %q", p.Identifier, "test@example.com")
		}
	})
}
