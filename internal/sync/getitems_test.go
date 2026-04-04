package sync

import (
	"context"
	"testing"
	"time"

	"github.com/danpicton/fauxtes-server/internal/domain"
)

func TestGetItems(t *testing.T) {
	t.Parallel()

	userUUID := "a4b8e3c2-1f6d-4a3b-9c7e-8d2f5a6b3c4d"

	makeItem := func(uuid string, ts int64) *domain.Item {
		return &domain.Item{
			UUID:               uuid,
			UserUUID:           userUUID,
			ContentType:        strPtr("Note"),
			UpdatedAtTimestamp: ts,
			CreatedAtTimestamp: ts,
			CreatedAt:          time.Unix(0, ts*1000), // micro to nano
			UpdatedAt:          time.Unix(0, ts*1000),
		}
	}

	t.Run("returns_items_since_last_sync", func(t *testing.T) {
		t.Parallel()
		repo := newFakeItemRepo()
		repo.addItem(makeItem("00000000-0000-0000-0000-000000000001", 100))
		repo.addItem(makeItem("00000000-0000-0000-0000-000000000002", 200))
		repo.addItem(makeItem("00000000-0000-0000-0000-000000000003", 300))

		syncToken := domain.EncodeSyncToken(150)
		resp, err := GetItems(context.Background(), repo, GetItemsRequest{
			UserUUID:  userUUID,
			SyncToken: syncToken,
		})
		if err != nil {
			t.Fatalf("GetItems() error = %v", err)
		}
		if len(resp.Items) != 2 {
			t.Errorf("got %d items, want 2", len(resp.Items))
		}
	})

	t.Run("generates_cursor_token_when_more_items", func(t *testing.T) {
		t.Parallel()
		repo := newFakeItemRepo()
		for i := 0; i < 160; i++ {
			repo.addItem(makeItem(
				fakeUUID(i),
				int64(1000+i),
			))
		}

		resp, err := GetItems(context.Background(), repo, GetItemsRequest{
			UserUUID: userUUID,
		})
		if err != nil {
			t.Fatalf("GetItems() error = %v", err)
		}
		if len(resp.Items) != 150 {
			t.Errorf("got %d items, want 150", len(resp.Items))
		}
		if resp.CursorToken == "" {
			t.Error("cursor token is empty, expected pagination cursor")
		}
	})

	t.Run("consumes_cursor_token_inclusive", func(t *testing.T) {
		t.Parallel()
		repo := newFakeItemRepo()
		repo.addItem(makeItem("00000000-0000-0000-0000-000000000001", 100))
		repo.addItem(makeItem("00000000-0000-0000-0000-000000000002", 200))
		repo.addItem(makeItem("00000000-0000-0000-0000-000000000003", 300))

		cursorToken := domain.EncodeSyncToken(200)
		resp, err := GetItems(context.Background(), repo, GetItemsRequest{
			UserUUID:    userUUID,
			CursorToken: cursorToken,
		})
		if err != nil {
			t.Fatalf("GetItems() error = %v", err)
		}
		// Should include item at ts=200 (inclusive) and ts=300
		if len(resp.Items) != 2 {
			t.Errorf("got %d items, want 2 (inclusive cursor)", len(resp.Items))
		}
	})

	t.Run("parses_v2_sync_token", func(t *testing.T) {
		t.Parallel()
		repo := newFakeItemRepo()
		repo.addItem(makeItem("00000000-0000-0000-0000-000000000001", 1000))

		syncToken := domain.EncodeSyncToken(500)
		resp, err := GetItems(context.Background(), repo, GetItemsRequest{
			UserUUID:  userUUID,
			SyncToken: syncToken,
		})
		if err != nil {
			t.Fatalf("GetItems() error = %v", err)
		}
		if len(resp.Items) != 1 {
			t.Errorf("got %d items, want 1", len(resp.Items))
		}
	})

	t.Run("invalid_sync_token_returns_error", func(t *testing.T) {
		t.Parallel()
		repo := newFakeItemRepo()

		_, err := GetItems(context.Background(), repo, GetItemsRequest{
			UserUUID:  userUUID,
			SyncToken: "!!!invalid!!!",
		})
		if err == nil {
			t.Error("expected error for invalid sync token")
		}
	})

	t.Run("enforces_upper_bound_limit", func(t *testing.T) {
		t.Parallel()
		repo := newFakeItemRepo()
		for i := 0; i < 200; i++ {
			repo.addItem(makeItem(fakeUUID(i), int64(1000+i)))
		}

		resp, err := GetItems(context.Background(), repo, GetItemsRequest{
			UserUUID: userUUID,
			Limit:    200, // exceeds MaxSyncLimit
		})
		if err != nil {
			t.Fatalf("GetItems() error = %v", err)
		}
		if len(resp.Items) != 150 {
			t.Errorf("got %d items, want 150 (capped)", len(resp.Items))
		}
	})

	t.Run("invalid_user_uuid_returns_error", func(t *testing.T) {
		t.Parallel()
		repo := newFakeItemRepo()

		_, err := GetItems(context.Background(), repo, GetItemsRequest{
			UserUUID: "not-a-uuid",
		})
		if err == nil {
			t.Error("expected error for invalid user UUID")
		}
	})

	t.Run("content_type_filter", func(t *testing.T) {
		t.Parallel()
		repo := newFakeItemRepo()
		noteItem := makeItem("00000000-0000-0000-0000-000000000001", 100)
		noteItem.ContentType = strPtr("Note")
		tagItem := makeItem("00000000-0000-0000-0000-000000000002", 200)
		tagItem.ContentType = strPtr("Tag")
		repo.addItem(noteItem)
		repo.addItem(tagItem)

		resp, err := GetItems(context.Background(), repo, GetItemsRequest{
			UserUUID:    userUUID,
			ContentType: "Note",
		})
		if err != nil {
			t.Fatalf("GetItems() error = %v", err)
		}
		if len(resp.Items) != 1 {
			t.Errorf("got %d items, want 1", len(resp.Items))
		}
	})

	t.Run("first_sync_no_token_returns_all", func(t *testing.T) {
		t.Parallel()
		repo := newFakeItemRepo()
		repo.addItem(makeItem("00000000-0000-0000-0000-000000000001", 100))
		repo.addItem(makeItem("00000000-0000-0000-0000-000000000002", 200))

		resp, err := GetItems(context.Background(), repo, GetItemsRequest{
			UserUUID: userUUID,
		})
		if err != nil {
			t.Fatalf("GetItems() error = %v", err)
		}
		if len(resp.Items) != 2 {
			t.Errorf("got %d items, want 2", len(resp.Items))
		}
	})
}
