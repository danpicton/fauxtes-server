package sync

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/danpicton/fauxtes-server/internal/domain"
	"github.com/google/uuid"
)

type SaveItemsRequest struct {
	UserUUID     string
	Items        []domain.ItemHash
	SessionUUID  string
	ReadOnly     bool
}

type SaveItemsResponse struct {
	SavedItems []*domain.Item
	Conflicts  []domain.SyncConflict
	SyncToken  string
}

func SaveItems(ctx context.Context, repo ItemRepository, req SaveItemsRequest) (*SaveItemsResponse, error) {
	if req.ReadOnly {
		return nil, fmt.Errorf("read-only access: cannot save items")
	}

	resp := &SaveItemsResponse{}

	if len(req.Items) == 0 {
		return resp, nil
	}

	var maxTimestamp int64

	for i := range req.Items {
		itemHash := &req.Items[i]

		// Validate UUID
		if err := itemHash.Validate(); err != nil {
			resp.Conflicts = append(resp.Conflicts, domain.SyncConflict{
				Type:        "uuid_conflict",
				UnsavedItem: itemHash,
			})
			continue
		}

		existing, err := repo.FindByUUID(ctx, itemHash.UUID)
		if err != nil {
			resp.Conflicts = append(resp.Conflicts, domain.SyncConflict{
				Type:        "uuid_conflict",
				UnsavedItem: itemHash,
			})
			continue
		}

		// Check for sync conflict BEFORE modifying: server has newer version
		if existing != nil && itemHash.UpdatedAtTimestamp != nil {
			if existing.UpdatedAtTimestamp > *itemHash.UpdatedAtTimestamp {
				serverRep := existing.ToHTTPRepresentation()
				resp.Conflicts = append(resp.Conflicts, domain.SyncConflict{
					Type:       "sync_conflict",
					ServerItem: &serverRep,
				})
				continue
			}
		}

		var item *domain.Item
		if existing != nil {
			item, err = updateExistingItem(existing, itemHash, req.SessionUUID)
		} else {
			item, err = createNewItem(itemHash, req.UserUUID, req.SessionUUID)
		}

		if err != nil {
			resp.Conflicts = append(resp.Conflicts, domain.SyncConflict{
				Type:        "uuid_conflict",
				UnsavedItem: itemHash,
			})
			continue
		}

		if err := repo.Save(ctx, item); err != nil {
			resp.Conflicts = append(resp.Conflicts, domain.SyncConflict{
				Type:        "uuid_conflict",
				UnsavedItem: itemHash,
			})
			continue
		}

		resp.SavedItems = append(resp.SavedItems, item)
		if item.UpdatedAtTimestamp > maxTimestamp {
			maxTimestamp = item.UpdatedAtTimestamp
		}
	}

	if maxTimestamp > 0 {
		// Add 1 microsecond to prevent sync doubles
		resp.SyncToken = domain.EncodeSyncToken(maxTimestamp + 1)
	}

	return resp, nil
}

func createNewItem(hash *domain.ItemHash, userUUID, sessionUUID string) (*domain.Item, error) {
	now := time.Now().UTC()
	nowMicro := now.UnixMicro()

	item := &domain.Item{
		UUID:     hash.UUID,
		UserUUID: userUUID,
	}

	applyHashToItem(item, hash)

	if hash.CreatedAtTimestamp != nil {
		item.CreatedAtTimestamp = *hash.CreatedAtTimestamp
	} else {
		item.CreatedAtTimestamp = nowMicro
	}
	item.UpdatedAtTimestamp = nowMicro

	if hash.CreatedAt != nil {
		t, err := time.Parse(time.RFC3339Nano, *hash.CreatedAt)
		if err == nil {
			item.CreatedAt = t
		} else {
			item.CreatedAt = now
		}
	} else {
		item.CreatedAt = now
	}
	item.UpdatedAt = now

	if sessionUUID != "" {
		item.UpdatedWithSession = &sessionUUID
	}

	item.ContentSize = calculateContentSize(hash)

	return item, nil
}

func updateExistingItem(existing *domain.Item, hash *domain.ItemHash, sessionUUID string) (*domain.Item, error) {
	now := time.Now().UTC()
	nowMicro := now.UnixMicro()

	item := existing
	applyHashToItem(item, hash)

	item.UpdatedAtTimestamp = nowMicro
	item.UpdatedAt = now

	if sessionUUID != "" {
		item.UpdatedWithSession = &sessionUUID
	}

	item.ContentSize = calculateContentSize(hash)

	return item, nil
}

func applyHashToItem(item *domain.Item, hash *domain.ItemHash) {
	item.Content = hash.Content
	item.ContentType = hash.ContentType
	item.EncItemKey = hash.EncItemKey
	item.AuthHash = hash.AuthHash
	item.ItemsKeyID = hash.ItemsKeyID
	item.DuplicateOf = hash.DuplicateOf
	item.KeySystemIdentifier = hash.KeySystemIdentifier
	item.SharedVaultUUID = hash.SharedVaultUUID

	if hash.Deleted != nil && *hash.Deleted {
		item.Deleted = true
		item.Content = nil
		item.EncItemKey = nil
		item.AuthHash = nil
	} else if hash.Deleted != nil {
		item.Deleted = *hash.Deleted
	}
}

func calculateContentSize(hash *domain.ItemHash) *int {
	data, _ := json.Marshal(hash)
	size := len(data)
	return &size
}

// ValidateItemHash checks if an item hash is valid for saving.
func ValidateItemHash(hash *domain.ItemHash) error {
	if _, err := uuid.Parse(hash.UUID); err != nil {
		return fmt.Errorf("invalid UUID: %w", err)
	}
	return nil
}
