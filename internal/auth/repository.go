package auth

import (
	"context"

	"github.com/danpicton/fauxtes-server/internal/domain"
)

type UserRepository interface {
	Create(ctx context.Context, user *domain.User) error
	FindByEmail(ctx context.Context, email string) (*domain.User, error)
	FindByUUID(ctx context.Context, uuid string) (*domain.User, error)
	Update(ctx context.Context, user *domain.User) error
}

type SessionRepository interface {
	Create(ctx context.Context, session *domain.Session) error
	FindByUUID(ctx context.Context, uuid string) (*domain.Session, error)
	FindByAccessToken(ctx context.Context, hashedToken string) (*domain.Session, error)
	FindByRefreshToken(ctx context.Context, hashedToken string) (*domain.Session, error)
	Delete(ctx context.Context, uuid string) error
	DeleteAllForUser(ctx context.Context, userUUID string) error
	Update(ctx context.Context, session *domain.Session) error
}
