package sync

import (
	"context"
	"fmt"

	"github.com/danpicton/fauxtes-server/internal/domain"
	"github.com/google/uuid"
)

const (
	MaxSyncLimit     = 150
	DefaultSyncLimit = 150
)

type GetItemsRequest struct {
	UserUUID    string
	SyncToken   string
	CursorToken string
	Limit       int
	ContentType string
}

type GetItemsResponse struct {
	Items       []*domain.Item
	CursorToken string
}

func GetItems(ctx context.Context, repo ItemRepository, req GetItemsRequest) (*GetItemsResponse, error) {
	if _, err := uuid.Parse(req.UserUUID); err != nil {
		return nil, fmt.Errorf("invalid user UUID: %w", err)
	}

	limit := req.Limit
	if limit <= 0 || limit > MaxSyncLimit {
		limit = DefaultSyncLimit
	}

	opts := FindItemsOpts{
		ContentType: req.ContentType,
		Limit:       limit + 1, // fetch one extra to detect if more items exist
	}

	if req.CursorToken != "" {
		ts, err := domain.DecodeSyncToken(req.CursorToken)
		if err != nil {
			return nil, fmt.Errorf("invalid cursor token: %w", err)
		}
		opts.FromTimestamp = ts
	} else if req.SyncToken != "" {
		ts, err := domain.DecodeSyncToken(req.SyncToken)
		if err != nil {
			return nil, fmt.Errorf("invalid sync token: %w", err)
		}
		opts.SinceTimestamp = ts
	}

	items, err := repo.FindByUserUUID(ctx, req.UserUUID, opts)
	if err != nil {
		return nil, fmt.Errorf("finding items: %w", err)
	}

	resp := &GetItemsResponse{}

	if len(items) > limit {
		// More items available - return cursor token
		resp.Items = items[:limit]
		lastItem := resp.Items[len(resp.Items)-1]
		resp.CursorToken = domain.EncodeSyncToken(lastItem.UpdatedAtTimestamp)
	} else {
		resp.Items = items
	}

	return resp, nil
}
