package domain

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
)

// Item represents a stored item in the database.
type Item struct {
	UUID                string
	UserUUID            string
	Content             *string
	ContentType         *string
	ContentSize         *int
	EncItemKey          *string
	AuthHash            *string
	ItemsKeyID          *string
	DuplicateOf         *string
	LastEditedBy        *string
	UpdatedWithSession  *string
	Deleted             bool
	SharedVaultUUID     *string
	KeySystemIdentifier *string
	CreatedAt           time.Time
	UpdatedAt           time.Time
	CreatedAtTimestamp  int64
	UpdatedAtTimestamp  int64
}

// ItemHTTPRepresentation is the JSON wire format returned to clients.
type ItemHTTPRepresentation struct {
	UUID               string  `json:"uuid"`
	ItemsKeyID         *string `json:"items_key_id"`
	DuplicateOf        *string `json:"duplicate_of"`
	EncItemKey         *string `json:"enc_item_key"`
	Content            *string `json:"content"`
	ContentType        string  `json:"content_type"`
	AuthHash           *string `json:"auth_hash"`
	Deleted            bool    `json:"deleted"`
	CreatedAt          string  `json:"created_at"`
	CreatedAtTimestamp int64   `json:"created_at_timestamp"`
	UpdatedAt          string  `json:"updated_at"`
	UpdatedAtTimestamp int64   `json:"updated_at_timestamp"`
	UpdatedWithSession *string `json:"updated_with_session"`
	KeySystemIdentifier *string `json:"key_system_identifier"`
	SharedVaultUUID    *string `json:"shared_vault_uuid"`
	UserUUID           *string `json:"user_uuid"`
	LastEditedByUUID   *string `json:"last_edited_by_uuid"`
}

// ItemHash is the format clients send items in during sync.
type ItemHash struct {
	UUID                string  `json:"uuid"`
	UserUUID            string  `json:"user_uuid,omitempty"`
	SharedVaultUUID     *string `json:"shared_vault_uuid"`
	Content             *string `json:"content"`
	ContentType         *string `json:"content_type"`
	AuthHash            *string `json:"auth_hash"`
	EncItemKey          *string `json:"enc_item_key"`
	ItemsKeyID          *string `json:"items_key_id"`
	KeySystemIdentifier *string `json:"key_system_identifier"`
	Deleted             *bool   `json:"deleted"`
	DuplicateOf         *string `json:"duplicate_of"`
	CreatedAt           *string `json:"created_at"`
	CreatedAtTimestamp  *int64  `json:"created_at_timestamp"`
	UpdatedAt           *string `json:"updated_at"`
	UpdatedAtTimestamp  *int64  `json:"updated_at_timestamp"`
}

// SyncConflict represents a conflict detected during sync.
type SyncConflict struct {
	Type       string                  `json:"type"`
	ServerItem *ItemHTTPRepresentation `json:"server_item,omitempty"`
	UnsavedItem *ItemHash              `json:"unsaved_item,omitempty"`
}

// Validate checks that the ItemHash has a valid UUID.
func (h *ItemHash) Validate() error {
	if h.UUID == "" {
		return fmt.Errorf("item UUID is required")
	}
	_, err := uuid.Parse(h.UUID)
	if err != nil {
		return fmt.Errorf("invalid item UUID %q: %w", h.UUID, err)
	}
	return nil
}

// ContentSize returns the approximate byte size of the item hash content.
func (h *ItemHash) ContentSize() int {
	data, _ := json.Marshal(h)
	return len(data)
}

// ToHTTPRepresentation converts an Item to its HTTP wire format.
func (item *Item) ToHTTPRepresentation() ItemHTTPRepresentation {
	ct := ""
	if item.ContentType != nil {
		ct = *item.ContentType
	}

	rep := ItemHTTPRepresentation{
		UUID:               item.UUID,
		ItemsKeyID:         item.ItemsKeyID,
		DuplicateOf:        item.DuplicateOf,
		EncItemKey:         item.EncItemKey,
		Content:            item.Content,
		ContentType:        ct,
		AuthHash:           item.AuthHash,
		Deleted:            item.Deleted,
		CreatedAt:          item.CreatedAt.UTC().Format(time.RFC3339Nano),
		CreatedAtTimestamp: item.CreatedAtTimestamp,
		UpdatedAt:          item.UpdatedAt.UTC().Format(time.RFC3339Nano),
		UpdatedAtTimestamp: item.UpdatedAtTimestamp,
		UpdatedWithSession: item.UpdatedWithSession,
		KeySystemIdentifier: item.KeySystemIdentifier,
		SharedVaultUUID:    item.SharedVaultUUID,
		LastEditedByUUID:   item.LastEditedBy,
	}

	if item.UserUUID != "" {
		rep.UserUUID = &item.UserUUID
	}

	return rep
}
