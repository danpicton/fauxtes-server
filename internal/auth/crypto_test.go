package auth

import (
	"testing"
)

func TestHashPassword(t *testing.T) {
	t.Parallel()

	hash, err := HashPassword("test-password-123")
	if err != nil {
		t.Fatalf("HashPassword() error = %v", err)
	}
	if hash == "" {
		t.Error("HashPassword() returned empty string")
	}
	if hash == "test-password-123" {
		t.Error("HashPassword() returned plaintext password")
	}
}

func TestCheckPassword(t *testing.T) {
	t.Parallel()

	hash, err := HashPassword("correct-password")
	if err != nil {
		t.Fatalf("HashPassword() error = %v", err)
	}

	tests := []struct {
		name     string
		password string
		want     bool
	}{
		{name: "correct_password", password: "correct-password", want: true},
		{name: "wrong_password", password: "wrong-password", want: false},
		{name: "empty_password", password: "", want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := CheckPassword(hash, tt.password)
			if got != tt.want {
				t.Errorf("CheckPassword(%q) = %v, want %v", tt.password, got, tt.want)
			}
		})
	}
}

func TestGenerateEncryptedServerKey(t *testing.T) {
	t.Parallel()

	key, err := GenerateEncryptedServerKey()
	if err != nil {
		t.Fatalf("GenerateEncryptedServerKey() error = %v", err)
	}
	if len(key) != 64 { // 32 bytes = 64 hex chars
		t.Errorf("GenerateEncryptedServerKey() length = %d, want 64", len(key))
	}

	// Should be different each time
	key2, _ := GenerateEncryptedServerKey()
	if key == key2 {
		t.Error("GenerateEncryptedServerKey() returned same key twice")
	}
}

func TestHashToken(t *testing.T) {
	t.Parallel()

	t.Run("deterministic", func(t *testing.T) {
		t.Parallel()
		h1 := HashToken("some-token")
		h2 := HashToken("some-token")
		if h1 != h2 {
			t.Errorf("HashToken not deterministic: %q != %q", h1, h2)
		}
	})

	t.Run("different_inputs", func(t *testing.T) {
		t.Parallel()
		h1 := HashToken("token-a")
		h2 := HashToken("token-b")
		if h1 == h2 {
			t.Error("HashToken produced same hash for different inputs")
		}
	})
}

func TestGenerateSessionTokens(t *testing.T) {
	t.Parallel()

	access, refresh, err := GenerateSessionTokens()
	if err != nil {
		t.Fatalf("GenerateSessionTokens() error = %v", err)
	}
	if access == "" || refresh == "" {
		t.Error("GenerateSessionTokens() returned empty tokens")
	}
	if access == refresh {
		t.Error("GenerateSessionTokens() returned same access and refresh tokens")
	}
}

func TestValidateCodeVerifier(t *testing.T) {
	t.Parallel()

	// Generate a challenge from a known verifier
	verifier := "test-code-verifier-12345"
	challenge := GenerateCodeChallenge(verifier)

	tests := []struct {
		name      string
		verifier  string
		challenge string
		want      bool
	}{
		{name: "valid_verifier", verifier: verifier, challenge: challenge, want: true},
		{name: "wrong_verifier", verifier: "wrong-verifier", challenge: challenge, want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := ValidateCodeVerifier(tt.verifier, tt.challenge)
			if got != tt.want {
				t.Errorf("ValidateCodeVerifier(%q, %q) = %v, want %v", tt.verifier, tt.challenge, got, tt.want)
			}
		})
	}
}
