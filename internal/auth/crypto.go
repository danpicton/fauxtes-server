package auth

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"

	"golang.org/x/crypto/bcrypt"
)

const bcryptCost = 10

func HashPassword(password string) (string, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcryptCost)
	if err != nil {
		return "", err
	}
	return string(hash), nil
}

func CheckPassword(hash, password string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
	return err == nil
}

func GenerateEncryptedServerKey() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

// HashToken produces a SHA-256 hex digest of a token string.
func HashToken(token string) string {
	h := sha256.Sum256([]byte(token))
	return hex.EncodeToString(h[:])
}

// GenerateSessionTokens creates random access and refresh token strings.
func GenerateSessionTokens() (accessToken, refreshToken string, err error) {
	access := make([]byte, 32)
	refresh := make([]byte, 32)
	if _, err := rand.Read(access); err != nil {
		return "", "", err
	}
	if _, err := rand.Read(refresh); err != nil {
		return "", "", err
	}
	return hex.EncodeToString(access), hex.EncodeToString(refresh), nil
}

// GenerateCodeChallenge creates a PKCE code challenge from a code verifier.
// challenge = base64url(sha256(verifier))
func GenerateCodeChallenge(verifier string) string {
	h := sha256.Sum256([]byte(verifier))
	return base64.RawURLEncoding.EncodeToString(h[:])
}

// ValidateCodeVerifier checks that sha256(verifier) matches the stored challenge.
func ValidateCodeVerifier(verifier, challenge string) bool {
	return GenerateCodeChallenge(verifier) == challenge
}
