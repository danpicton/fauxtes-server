package sqlite

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/danpicton/fauxtes-server/internal/domain"
)

type SessionRepo struct {
	db *sql.DB
}

func NewSessionRepo(db *sql.DB) *SessionRepo {
	return &SessionRepo{db: db}
}

func (r *SessionRepo) Create(ctx context.Context, s *domain.Session) error {
	_, err := r.db.ExecContext(ctx, `
		INSERT INTO sessions (uuid, user_uuid, private_identifier, hashed_access_token,
			hashed_refresh_token, access_expiration, refresh_expiration, api_version,
			user_agent, readonly_access, version, application, snjs, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		s.UUID, s.UserUUID, s.PrivateIdentifier, s.HashedAccessToken,
		s.HashedRefreshToken, s.AccessExpiration, s.RefreshExpiration, s.APIVersion,
		s.UserAgent, s.ReadonlyAccess, s.Version, s.Application, s.SNJS,
		s.CreatedAt, s.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("inserting session: %w", err)
	}
	return nil
}

func (r *SessionRepo) FindByUUID(ctx context.Context, uuid string) (*domain.Session, error) {
	return r.scanSession(r.db.QueryRowContext(ctx, `SELECT * FROM sessions WHERE uuid = ?`, uuid))
}

func (r *SessionRepo) FindByAccessToken(ctx context.Context, hashedToken string) (*domain.Session, error) {
	return r.scanSession(r.db.QueryRowContext(ctx, `SELECT * FROM sessions WHERE hashed_access_token = ?`, hashedToken))
}

func (r *SessionRepo) FindByRefreshToken(ctx context.Context, hashedToken string) (*domain.Session, error) {
	return r.scanSession(r.db.QueryRowContext(ctx, `SELECT * FROM sessions WHERE hashed_refresh_token = ?`, hashedToken))
}

func (r *SessionRepo) Delete(ctx context.Context, uuid string) error {
	_, err := r.db.ExecContext(ctx, `DELETE FROM sessions WHERE uuid = ?`, uuid)
	return err
}

func (r *SessionRepo) DeleteAllForUser(ctx context.Context, userUUID string) error {
	_, err := r.db.ExecContext(ctx, `DELETE FROM sessions WHERE user_uuid = ?`, userUUID)
	return err
}

func (r *SessionRepo) Update(ctx context.Context, s *domain.Session) error {
	_, err := r.db.ExecContext(ctx, `
		UPDATE sessions SET user_uuid=?, private_identifier=?, hashed_access_token=?,
			hashed_refresh_token=?, access_expiration=?, refresh_expiration=?, api_version=?,
			user_agent=?, readonly_access=?, version=?, application=?, snjs=?, updated_at=?
		WHERE uuid=?`,
		s.UserUUID, s.PrivateIdentifier, s.HashedAccessToken,
		s.HashedRefreshToken, s.AccessExpiration, s.RefreshExpiration, s.APIVersion,
		s.UserAgent, s.ReadonlyAccess, s.Version, s.Application, s.SNJS,
		s.UpdatedAt, s.UUID,
	)
	return err
}

func (r *SessionRepo) scanSession(row *sql.Row) (*domain.Session, error) {
	s := &domain.Session{}
	err := row.Scan(
		&s.UUID, &s.UserUUID, &s.PrivateIdentifier, &s.HashedAccessToken,
		&s.HashedRefreshToken, &s.AccessExpiration, &s.RefreshExpiration, &s.APIVersion,
		&s.UserAgent, &s.ReadonlyAccess, &s.Version, &s.Application, &s.SNJS,
		&s.CreatedAt, &s.UpdatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("scanning session: %w", err)
	}
	return s, nil
}
