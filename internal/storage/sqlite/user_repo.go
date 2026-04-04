package sqlite

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/danpicton/fauxtes-server/internal/domain"
)

type UserRepo struct {
	db *sql.DB
}

func NewUserRepo(db *sql.DB) *UserRepo {
	return &UserRepo{db: db}
}

func (r *UserRepo) Create(ctx context.Context, u *domain.User) error {
	_, err := r.db.ExecContext(ctx, `
		INSERT INTO users (uuid, email, encrypted_password, pw_nonce, pw_salt, pw_cost,
			pw_key_size, pw_alg, pw_func, encrypted_server_key, server_encryption_version,
			version, kp_created, kp_origination, locked_until, num_failed_attempts,
			created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		u.UUID, u.Email, u.EncryptedPassword, u.PwNonce, u.PwSalt, u.PwCost,
		u.PwKeySize, u.PwAlg, u.PwFunc, u.EncryptedServerKey, u.ServerEncryptionVersion,
		u.Version, u.KpCreated, u.KpOrigination, u.LockedUntil, u.NumFailedAttempts,
		u.CreatedAt, u.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("inserting user: %w", err)
	}
	return nil
}

func (r *UserRepo) FindByEmail(ctx context.Context, email string) (*domain.User, error) {
	return r.scanUser(r.db.QueryRowContext(ctx, `SELECT * FROM users WHERE email = ?`, email))
}

func (r *UserRepo) FindByUUID(ctx context.Context, uuid string) (*domain.User, error) {
	return r.scanUser(r.db.QueryRowContext(ctx, `SELECT * FROM users WHERE uuid = ?`, uuid))
}

func (r *UserRepo) Update(ctx context.Context, u *domain.User) error {
	_, err := r.db.ExecContext(ctx, `
		UPDATE users SET email=?, encrypted_password=?, pw_nonce=?, pw_salt=?, pw_cost=?,
			pw_key_size=?, pw_alg=?, pw_func=?, encrypted_server_key=?, server_encryption_version=?,
			version=?, kp_created=?, kp_origination=?, locked_until=?, num_failed_attempts=?,
			updated_at=?
		WHERE uuid=?`,
		u.Email, u.EncryptedPassword, u.PwNonce, u.PwSalt, u.PwCost,
		u.PwKeySize, u.PwAlg, u.PwFunc, u.EncryptedServerKey, u.ServerEncryptionVersion,
		u.Version, u.KpCreated, u.KpOrigination, u.LockedUntil, u.NumFailedAttempts,
		u.UpdatedAt, u.UUID,
	)
	return err
}

func (r *UserRepo) scanUser(row *sql.Row) (*domain.User, error) {
	u := &domain.User{}
	err := row.Scan(
		&u.UUID, &u.Email, &u.EncryptedPassword, &u.PwNonce, &u.PwSalt, &u.PwCost,
		&u.PwKeySize, &u.PwAlg, &u.PwFunc, &u.EncryptedServerKey, &u.ServerEncryptionVersion,
		&u.Version, &u.KpCreated, &u.KpOrigination, &u.LockedUntil, &u.NumFailedAttempts,
		&u.CreatedAt, &u.UpdatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("scanning user: %w", err)
	}
	return u, nil
}
