package domain

import (
	"crypto/sha256"
	"fmt"
	"time"

	"github.com/google/uuid"
)

type User struct {
	UUID                    string
	Email                   string
	EncryptedPassword       string
	PwNonce                 *string
	PwSalt                  *string
	PwCost                  *int
	PwKeySize               *int
	PwAlg                   *string
	PwFunc                  *string
	EncryptedServerKey      *string
	ServerEncryptionVersion int
	Version                 *string
	KpCreated               *string
	KpOrigination           *string
	LockedUntil             *time.Time
	NumFailedAttempts       *int
	CreatedAt               time.Time
	UpdatedAt               time.Time
}

// KeyParams represents the key derivation parameters returned to clients.
type KeyParams struct {
	Identifier  string `json:"identifier"`
	PwNonce     string `json:"pw_nonce,omitempty"`
	Version     string `json:"version"`
	PwCost      int    `json:"pw_cost,omitempty"`
	PwSalt      string `json:"pw_salt,omitempty"`
	PwAlg       string `json:"pw_alg,omitempty"`
	PwFunc      string `json:"pw_func,omitempty"`
	PwKeySize   int    `json:"pw_key_size,omitempty"`
	Created     string `json:"created,omitempty"`
	Origination string `json:"origination,omitempty"`
}

func NewUser(email string) User {
	now := time.Now().UTC()
	return User{
		UUID:      uuid.New().String(),
		Email:     email,
		CreatedAt: now,
		UpdatedAt: now,
	}
}

// PseudoKeyParams generates deterministic key params for a non-existent user.
// This prevents email enumeration attacks by returning plausible params
// regardless of whether the account exists.
func PseudoKeyParams(email string) KeyParams {
	h := sha256.Sum256([]byte(email + "pseudo_nonce_salt"))
	nonce := fmt.Sprintf("%x", h[:32])
	return KeyParams{
		Identifier: email,
		PwNonce:    nonce,
		Version:    "004",
	}
}

// KeyParamsFromUser generates key params from an existing user.
func KeyParamsFromUser(u User, authenticated bool) KeyParams {
	version := "004"
	if u.Version != nil {
		version = *u.Version
	}

	kp := KeyParams{
		Identifier: u.Email,
		Version:    version,
	}

	switch version {
	case "004":
		if u.PwNonce != nil {
			kp.PwNonce = *u.PwNonce
		}
		if authenticated {
			if u.KpCreated != nil {
				kp.Created = *u.KpCreated
			}
			if u.KpOrigination != nil {
				kp.Origination = *u.KpOrigination
			}
		}
	case "003":
		if u.PwNonce != nil {
			kp.PwNonce = *u.PwNonce
		}
		if u.PwCost != nil {
			kp.PwCost = *u.PwCost
		}
	case "002":
		if u.PwSalt != nil {
			kp.PwSalt = *u.PwSalt
		}
		if u.PwCost != nil {
			kp.PwCost = *u.PwCost
		}
	case "001":
		if u.PwSalt != nil {
			kp.PwSalt = *u.PwSalt
		}
		if u.PwCost != nil {
			kp.PwCost = *u.PwCost
		}
		if u.PwAlg != nil {
			kp.PwAlg = *u.PwAlg
		}
		if u.PwFunc != nil {
			kp.PwFunc = *u.PwFunc
		}
		if u.PwKeySize != nil {
			kp.PwKeySize = *u.PwKeySize
		}
	}

	return kp
}
