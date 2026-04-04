package domain

import (
	"time"

	"github.com/google/uuid"
)

type Session struct {
	UUID               string
	UserUUID           string
	PrivateIdentifier  string
	HashedAccessToken  string
	HashedRefreshToken string
	AccessExpiration   time.Time
	RefreshExpiration  time.Time
	APIVersion         *string
	UserAgent          *string
	ReadonlyAccess     bool
	Version            int
	Application        *string
	SNJS               *string
	CreatedAt          time.Time
	UpdatedAt          time.Time
}

func NewSession(userUUID string) Session {
	now := time.Now().UTC()
	return Session{
		UUID:              uuid.New().String(),
		UserUUID:          userUUID,
		PrivateIdentifier: uuid.New().String(),
		Version:           1,
		CreatedAt:         now,
		UpdatedAt:         now,
	}
}
