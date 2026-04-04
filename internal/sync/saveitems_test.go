package sync

import (
	"context"
	"testing"
	"time"

	"github.com/danpicton/fauxtes-server/internal/domain"
)

func TestSaveItems(t *testing.T) {
	t.Parallel()

	userUUID := "a4b8e3c2-1f6d-4a3b-9c7e-8d2f5a6b3c4d"
	sessionUUID := "b5c9f4d3-2e7a-4b8c-ad6f-9e3a7b4c5d6e"

	t.Run("new_item_creates", func(t *testing.T) {
		t.Parallel()
		repo := newFakeItemRepo()
		resp, err := SaveItems(context.Background(), repo, SaveItemsRequest{
			UserUUID:    userUUID,
			SessionUUID: sessionUUID,
			Items: []domain.ItemHash{
				{UUID: "00000000-0000-0000-0000-000000000001", Content: strPtr("enc-content"), ContentType: strPtr("Note")},
			},
		})
		if err != nil {
			t.Fatalf("SaveItems() error = %v", err)
		}
		if len(resp.SavedItems) != 1 {
			t.Fatalf("saved %d items, want 1", len(resp.SavedItems))
		}
		if resp.SavedItems[0].UUID != "00000000-0000-0000-0000-000000000001" {
			t.Errorf("UUID = %q, want %q", resp.SavedItems[0].UUID, "00000000-0000-0000-0000-000000000001")
		}
		if len(resp.Conflicts) != 0 {
			t.Errorf("conflicts = %d, want 0", len(resp.Conflicts))
		}
	})

	t.Run("existing_item_updates", func(t *testing.T) {
		t.Parallel()
		repo := newFakeItemRepo()
		existing := &domain.Item{
			UUID:               "00000000-0000-0000-0000-000000000001",
			UserUUID:           userUUID,
			Content:            strPtr("old-content"),
			ContentType:        strPtr("Note"),
			UpdatedAtTimestamp: 100,
			CreatedAtTimestamp: 50,
			CreatedAt:          time.Now().UTC(),
			UpdatedAt:          time.Now().UTC(),
		}
		repo.addItem(existing)

		ts := int64(100) // same as existing so no conflict
		resp, err := SaveItems(context.Background(), repo, SaveItemsRequest{
			UserUUID:    userUUID,
			SessionUUID: sessionUUID,
			Items: []domain.ItemHash{
				{UUID: "00000000-0000-0000-0000-000000000001", Content: strPtr("new-content"), ContentType: strPtr("Note"), UpdatedAtTimestamp: &ts},
			},
		})
		if err != nil {
			t.Fatalf("SaveItems() error = %v", err)
		}
		if len(resp.SavedItems) != 1 {
			t.Fatalf("saved %d items, want 1", len(resp.SavedItems))
		}
		saved := repo.items["00000000-0000-0000-0000-000000000001"]
		if saved.Content == nil || *saved.Content != "new-content" {
			t.Errorf("content not updated")
		}
	})

	t.Run("invalid_uuid_returns_conflict", func(t *testing.T) {
		t.Parallel()
		repo := newFakeItemRepo()
		resp, err := SaveItems(context.Background(), repo, SaveItemsRequest{
			UserUUID: userUUID,
			Items: []domain.ItemHash{
				{UUID: "not-a-valid-uuid", Content: strPtr("content")},
			},
		})
		if err != nil {
			t.Fatalf("SaveItems() error = %v", err)
		}
		if len(resp.Conflicts) != 1 {
			t.Fatalf("conflicts = %d, want 1", len(resp.Conflicts))
		}
		if resp.Conflicts[0].Type != "uuid_conflict" {
			t.Errorf("conflict type = %q, want %q", resp.Conflicts[0].Type, "uuid_conflict")
		}
	})

	t.Run("readonly_session_blocks_saves", func(t *testing.T) {
		t.Parallel()
		repo := newFakeItemRepo()
		_, err := SaveItems(context.Background(), repo, SaveItemsRequest{
			UserUUID: userUUID,
			ReadOnly: true,
			Items: []domain.ItemHash{
				{UUID: "00000000-0000-0000-0000-000000000001"},
			},
		})
		if err == nil {
			t.Error("expected error for read-only session")
		}
	})

	t.Run("server_newer_returns_sync_conflict", func(t *testing.T) {
		t.Parallel()
		repo := newFakeItemRepo()
		existing := &domain.Item{
			UUID:               "00000000-0000-0000-0000-000000000001",
			UserUUID:           userUUID,
			Content:            strPtr("server-content"),
			ContentType:        strPtr("Note"),
			UpdatedAtTimestamp: 500,
			CreatedAtTimestamp: 100,
			CreatedAt:          time.Now().UTC(),
			UpdatedAt:          time.Now().UTC(),
		}
		repo.addItem(existing)

		clientTS := int64(200) // older than server's 500
		resp, err := SaveItems(context.Background(), repo, SaveItemsRequest{
			UserUUID: userUUID,
			Items: []domain.ItemHash{
				{UUID: "00000000-0000-0000-0000-000000000001", Content: strPtr("client-content"), ContentType: strPtr("Note"), UpdatedAtTimestamp: &clientTS},
			},
		})
		if err != nil {
			t.Fatalf("SaveItems() error = %v", err)
		}
		if len(resp.Conflicts) != 1 {
			t.Fatalf("conflicts = %d, want 1", len(resp.Conflicts))
		}
		if resp.Conflicts[0].Type != "sync_conflict" {
			t.Errorf("conflict type = %q, want %q", resp.Conflicts[0].Type, "sync_conflict")
		}
		if resp.Conflicts[0].ServerItem == nil {
			t.Error("sync_conflict should include server_item")
		}
	})

	t.Run("sync_token_from_latest_timestamp_plus_one", func(t *testing.T) {
		t.Parallel()
		repo := newFakeItemRepo()
		resp, err := SaveItems(context.Background(), repo, SaveItemsRequest{
			UserUUID: userUUID,
			Items: []domain.ItemHash{
				{UUID: "00000000-0000-0000-0000-000000000001", Content: strPtr("c1"), ContentType: strPtr("Note")},
				{UUID: "00000000-0000-0000-0000-000000000002", Content: strPtr("c2"), ContentType: strPtr("Note")},
			},
		})
		if err != nil {
			t.Fatalf("SaveItems() error = %v", err)
		}
		if resp.SyncToken == "" {
			t.Error("sync token is empty")
		}
		ts, err := domain.DecodeSyncToken(resp.SyncToken)
		if err != nil {
			t.Fatalf("decode sync token: %v", err)
		}
		// Sync token should be max(updated_at_timestamp) + 1
		var maxTS int64
		for _, item := range resp.SavedItems {
			if item.UpdatedAtTimestamp > maxTS {
				maxTS = item.UpdatedAtTimestamp
			}
		}
		if ts != maxTS+1 {
			t.Errorf("sync token timestamp = %d, want %d (max+1)", ts, maxTS+1)
		}
	})

	t.Run("content_size_calculated", func(t *testing.T) {
		t.Parallel()
		repo := newFakeItemRepo()
		resp, err := SaveItems(context.Background(), repo, SaveItemsRequest{
			UserUUID: userUUID,
			Items: []domain.ItemHash{
				{UUID: "00000000-0000-0000-0000-000000000001", Content: strPtr("some content"), ContentType: strPtr("Note")},
			},
		})
		if err != nil {
			t.Fatalf("SaveItems() error = %v", err)
		}
		if resp.SavedItems[0].ContentSize == nil || *resp.SavedItems[0].ContentSize <= 0 {
			t.Error("content size not calculated")
		}
	})

	t.Run("sets_updated_with_session", func(t *testing.T) {
		t.Parallel()
		repo := newFakeItemRepo()
		resp, err := SaveItems(context.Background(), repo, SaveItemsRequest{
			UserUUID:    userUUID,
			SessionUUID: sessionUUID,
			Items: []domain.ItemHash{
				{UUID: "00000000-0000-0000-0000-000000000001", ContentType: strPtr("Note")},
			},
		})
		if err != nil {
			t.Fatalf("SaveItems() error = %v", err)
		}
		if resp.SavedItems[0].UpdatedWithSession == nil || *resp.SavedItems[0].UpdatedWithSession != sessionUUID {
			t.Errorf("UpdatedWithSession = %v, want %q", resp.SavedItems[0].UpdatedWithSession, sessionUUID)
		}
	})

	t.Run("deleted_item_clears_content", func(t *testing.T) {
		t.Parallel()
		repo := newFakeItemRepo()
		resp, err := SaveItems(context.Background(), repo, SaveItemsRequest{
			UserUUID: userUUID,
			Items: []domain.ItemHash{
				{UUID: "00000000-0000-0000-0000-000000000001", Content: strPtr("content"), ContentType: strPtr("Note"), Deleted: boolPtr(true)},
			},
		})
		if err != nil {
			t.Fatalf("SaveItems() error = %v", err)
		}
		item := resp.SavedItems[0]
		if !item.Deleted {
			t.Error("item should be marked deleted")
		}
		if item.Content != nil {
			t.Errorf("deleted item content = %v, want nil", item.Content)
		}
		if item.EncItemKey != nil {
			t.Errorf("deleted item enc_item_key = %v, want nil", item.EncItemKey)
		}
	})

	t.Run("sets_timestamps", func(t *testing.T) {
		t.Parallel()
		repo := newFakeItemRepo()
		resp, err := SaveItems(context.Background(), repo, SaveItemsRequest{
			UserUUID: userUUID,
			Items: []domain.ItemHash{
				{UUID: "00000000-0000-0000-0000-000000000001", ContentType: strPtr("Note")},
			},
		})
		if err != nil {
			t.Fatalf("SaveItems() error = %v", err)
		}
		item := resp.SavedItems[0]
		if item.CreatedAtTimestamp == 0 {
			t.Error("CreatedAtTimestamp is zero")
		}
		if item.UpdatedAtTimestamp == 0 {
			t.Error("UpdatedAtTimestamp is zero")
		}
		if item.CreatedAt.IsZero() {
			t.Error("CreatedAt is zero")
		}
		if item.UpdatedAt.IsZero() {
			t.Error("UpdatedAt is zero")
		}
	})

	t.Run("empty_items_returns_empty", func(t *testing.T) {
		t.Parallel()
		repo := newFakeItemRepo()
		resp, err := SaveItems(context.Background(), repo, SaveItemsRequest{
			UserUUID: userUUID,
			Items:    []domain.ItemHash{},
		})
		if err != nil {
			t.Fatalf("SaveItems() error = %v", err)
		}
		if len(resp.SavedItems) != 0 {
			t.Errorf("saved %d items, want 0", len(resp.SavedItems))
		}
		if len(resp.Conflicts) != 0 {
			t.Errorf("conflicts = %d, want 0", len(resp.Conflicts))
		}
	})

	t.Run("multiple_items_batch", func(t *testing.T) {
		t.Parallel()
		repo := newFakeItemRepo()
		resp, err := SaveItems(context.Background(), repo, SaveItemsRequest{
			UserUUID: userUUID,
			Items: []domain.ItemHash{
				{UUID: "00000000-0000-0000-0000-000000000001", Content: strPtr("c1"), ContentType: strPtr("Note")},
				{UUID: "00000000-0000-0000-0000-000000000002", Content: strPtr("c2"), ContentType: strPtr("Tag")},
				{UUID: "not-valid", Content: strPtr("c3")}, // will conflict
			},
		})
		if err != nil {
			t.Fatalf("SaveItems() error = %v", err)
		}
		if len(resp.SavedItems) != 2 {
			t.Errorf("saved %d items, want 2", len(resp.SavedItems))
		}
		if len(resp.Conflicts) != 1 {
			t.Errorf("conflicts = %d, want 1", len(resp.Conflicts))
		}
	})
}
