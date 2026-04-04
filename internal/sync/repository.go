package sync

import (
	"context"

	"github.com/danpicton/fauxtes-server/internal/domain"
)

type FindItemsOpts struct {
	SinceTimestamp int64  // updated_at_timestamp > this (exclusive, for sync_token)
	FromTimestamp  int64  // updated_at_timestamp >= this (inclusive, for cursor)
	ContentType    string // optional filter
	Limit          int
}

type ItemRepository interface {
	FindByUUID(ctx context.Context, uuid string) (*domain.Item, error)
	FindByUserUUID(ctx context.Context, userUUID string, opts FindItemsOpts) ([]*domain.Item, error)
	Save(ctx context.Context, item *domain.Item) error // upsert
}
