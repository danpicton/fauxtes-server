package sqlite

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	"github.com/danpicton/fauxtes-server/internal/domain"
	syncpkg "github.com/danpicton/fauxtes-server/internal/sync"
)

type ItemRepo struct {
	db *sql.DB
}

func NewItemRepo(db *sql.DB) *ItemRepo {
	return &ItemRepo{db: db}
}

func (r *ItemRepo) FindByUUID(ctx context.Context, uuid string) (*domain.Item, error) {
	row := r.db.QueryRowContext(ctx, `SELECT * FROM items WHERE uuid = ?`, uuid)
	return r.scanItem(row)
}

func (r *ItemRepo) FindByUserUUID(ctx context.Context, userUUID string, opts syncpkg.FindItemsOpts) ([]*domain.Item, error) {
	var conditions []string
	var args []any

	conditions = append(conditions, "user_uuid = ?")
	args = append(args, userUUID)

	if opts.SinceTimestamp > 0 {
		conditions = append(conditions, "updated_at_timestamp > ?")
		args = append(args, opts.SinceTimestamp)
	}
	if opts.FromTimestamp > 0 {
		conditions = append(conditions, "updated_at_timestamp >= ?")
		args = append(args, opts.FromTimestamp)
	}
	if opts.ContentType != "" {
		conditions = append(conditions, "content_type = ?")
		args = append(args, opts.ContentType)
	}

	query := "SELECT * FROM items WHERE " + strings.Join(conditions, " AND ") + " ORDER BY updated_at_timestamp ASC"
	if opts.Limit > 0 {
		query += fmt.Sprintf(" LIMIT %d", opts.Limit)
	}

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("querying items: %w", err)
	}
	defer rows.Close()

	var items []*domain.Item
	for rows.Next() {
		item, err := r.scanItemFromRows(rows)
		if err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func (r *ItemRepo) Save(ctx context.Context, item *domain.Item) error {
	_, err := r.db.ExecContext(ctx, `
		INSERT INTO items (uuid, user_uuid, content, content_type, content_size, enc_item_key,
			auth_hash, items_key_id, duplicate_of, last_edited_by, updated_with_session,
			deleted, shared_vault_uuid, key_system_identifier, created_at, updated_at,
			created_at_timestamp, updated_at_timestamp)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(uuid) DO UPDATE SET
			content=excluded.content, content_type=excluded.content_type,
			content_size=excluded.content_size, enc_item_key=excluded.enc_item_key,
			auth_hash=excluded.auth_hash, items_key_id=excluded.items_key_id,
			duplicate_of=excluded.duplicate_of, last_edited_by=excluded.last_edited_by,
			updated_with_session=excluded.updated_with_session, deleted=excluded.deleted,
			shared_vault_uuid=excluded.shared_vault_uuid,
			key_system_identifier=excluded.key_system_identifier,
			updated_at=excluded.updated_at, updated_at_timestamp=excluded.updated_at_timestamp`,
		item.UUID, item.UserUUID, item.Content, item.ContentType, item.ContentSize,
		item.EncItemKey, item.AuthHash, item.ItemsKeyID, item.DuplicateOf,
		item.LastEditedBy, item.UpdatedWithSession, item.Deleted, item.SharedVaultUUID,
		item.KeySystemIdentifier, item.CreatedAt, item.UpdatedAt,
		item.CreatedAtTimestamp, item.UpdatedAtTimestamp,
	)
	if err != nil {
		return fmt.Errorf("saving item: %w", err)
	}
	return nil
}

func (r *ItemRepo) scanItem(row *sql.Row) (*domain.Item, error) {
	item := &domain.Item{}
	err := row.Scan(
		&item.UUID, &item.UserUUID, &item.Content, &item.ContentType, &item.ContentSize,
		&item.EncItemKey, &item.AuthHash, &item.ItemsKeyID, &item.DuplicateOf,
		&item.LastEditedBy, &item.UpdatedWithSession, &item.Deleted, &item.SharedVaultUUID,
		&item.KeySystemIdentifier, &item.CreatedAt, &item.UpdatedAt,
		&item.CreatedAtTimestamp, &item.UpdatedAtTimestamp,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("scanning item: %w", err)
	}
	return item, nil
}

func (r *ItemRepo) scanItemFromRows(rows *sql.Rows) (*domain.Item, error) {
	item := &domain.Item{}
	err := rows.Scan(
		&item.UUID, &item.UserUUID, &item.Content, &item.ContentType, &item.ContentSize,
		&item.EncItemKey, &item.AuthHash, &item.ItemsKeyID, &item.DuplicateOf,
		&item.LastEditedBy, &item.UpdatedWithSession, &item.Deleted, &item.SharedVaultUUID,
		&item.KeySystemIdentifier, &item.CreatedAt, &item.UpdatedAt,
		&item.CreatedAtTimestamp, &item.UpdatedAtTimestamp,
	)
	if err != nil {
		return nil, fmt.Errorf("scanning item row: %w", err)
	}
	return item, nil
}
