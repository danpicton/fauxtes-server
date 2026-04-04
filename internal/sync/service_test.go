package sync

import (
	"context"
	"testing"
	"time"

	"github.com/danpicton/fauxtes-server/internal/domain"
)

func TestSync(t *testing.T) {
	t.Parallel()

	userUUID := "a4b8e3c2-1f6d-4a3b-9c7e-8d2f5a6b3c4d"

	makeItem := func(uuid string, ts int64, ct string) *domain.Item {
		return &domain.Item{
			UUID:               uuid,
			UserUUID:           userUUID,
			Content:            strPtr("encrypted"),
			ContentType:        strPtr(ct),
			UpdatedAtTimestamp: ts,
			CreatedAtTimestamp: ts,
			CreatedAt:          time.Unix(0, ts*1000),
			UpdatedAt:          time.Unix(0, ts*1000),
		}
	}

	t.Run("basic_save_then_retrieve", func(t *testing.T) {
		t.Parallel()
		repo := newFakeItemRepo()
		svc := NewService(repo)

		resp, err := svc.Sync(context.Background(), SyncRequest{
			UserUUID: userUUID,
			Items: []domain.ItemHash{
				{UUID: "00000000-0000-0000-0000-000000000001", Content: strPtr("note1"), ContentType: strPtr("Note")},
			},
		})
		if err != nil {
			t.Fatalf("Sync() error = %v", err)
		}
		if len(resp.SavedItems) != 1 {
			t.Errorf("saved %d items, want 1", len(resp.SavedItems))
		}
		if resp.SyncToken == "" {
			t.Error("sync token is empty")
		}
	})

	t.Run("first_sync_returns_all_items", func(t *testing.T) {
		t.Parallel()
		repo := newFakeItemRepo()
		repo.addItem(makeItem("00000000-0000-0000-0000-000000000001", 100, "Note"))
		repo.addItem(makeItem("00000000-0000-0000-0000-000000000002", 200, "Note"))
		svc := NewService(repo)

		resp, err := svc.Sync(context.Background(), SyncRequest{UserUUID: userUUID})
		if err != nil {
			t.Fatalf("Sync() error = %v", err)
		}
		if len(resp.RetrievedItems) != 2 {
			t.Errorf("retrieved %d items, want 2", len(resp.RetrievedItems))
		}
	})

	t.Run("consecutive_sync_filters_just_saved", func(t *testing.T) {
		t.Parallel()
		repo := newFakeItemRepo()
		repo.addItem(makeItem("00000000-0000-0000-0000-000000000001", 100, "Note"))
		svc := NewService(repo)

		// Sync with a new item
		resp, err := svc.Sync(context.Background(), SyncRequest{
			UserUUID: userUUID,
			Items: []domain.ItemHash{
				{UUID: "00000000-0000-0000-0000-000000000002", Content: strPtr("new"), ContentType: strPtr("Note")},
			},
		})
		if err != nil {
			t.Fatalf("Sync() error = %v", err)
		}
		// The just-saved item should NOT appear in retrieved_items
		for _, item := range resp.RetrievedItems {
			if item.UUID == "00000000-0000-0000-0000-000000000002" {
				t.Error("just-saved item should not appear in retrieved_items")
			}
		}
	})

	t.Run("error_in_get_returns_error", func(t *testing.T) {
		t.Parallel()
		repo := newFakeItemRepo()
		svc := NewService(repo)

		_, err := svc.Sync(context.Background(), SyncRequest{
			UserUUID:  "not-a-valid-uuid", // will fail GetItems validation
			SyncToken: domain.EncodeSyncToken(0),
		})
		if err == nil {
			t.Error("expected error from GetItems")
		}
	})

	t.Run("returns_new_sync_token", func(t *testing.T) {
		t.Parallel()
		repo := newFakeItemRepo()
		repo.addItem(makeItem("00000000-0000-0000-0000-000000000001", 500, "Note"))
		svc := NewService(repo)

		resp, err := svc.Sync(context.Background(), SyncRequest{UserUUID: userUUID})
		if err != nil {
			t.Fatalf("Sync() error = %v", err)
		}
		if resp.SyncToken == "" {
			t.Error("sync token is empty")
		}
		ts, _ := domain.DecodeSyncToken(resp.SyncToken)
		if ts <= 0 {
			t.Errorf("sync token timestamp = %d, want > 0", ts)
		}
	})

	t.Run("returns_cursor_token_when_paginated", func(t *testing.T) {
		t.Parallel()
		repo := newFakeItemRepo()
		for i := 0; i < 160; i++ {
			repo.addItem(makeItem(fakeUUID(i), int64(1000+i), "Note"))
		}
		svc := NewService(repo)

		resp, err := svc.Sync(context.Background(), SyncRequest{UserUUID: userUUID})
		if err != nil {
			t.Fatalf("Sync() error = %v", err)
		}
		if resp.CursorToken == nil {
			t.Error("cursor token is nil, expected pagination cursor")
		}
	})

	t.Run("no_items_to_save_only_retrieves", func(t *testing.T) {
		t.Parallel()
		repo := newFakeItemRepo()
		repo.addItem(makeItem("00000000-0000-0000-0000-000000000001", 100, "Note"))
		svc := NewService(repo)

		resp, err := svc.Sync(context.Background(), SyncRequest{UserUUID: userUUID})
		if err != nil {
			t.Fatalf("Sync() error = %v", err)
		}
		if len(resp.SavedItems) != 0 {
			t.Errorf("saved %d items, want 0", len(resp.SavedItems))
		}
		if len(resp.RetrievedItems) != 1 {
			t.Errorf("retrieved %d items, want 1", len(resp.RetrievedItems))
		}
	})

	t.Run("compute_integrity_hash", func(t *testing.T) {
		t.Parallel()
		repo := newFakeItemRepo()
		repo.addItem(makeItem("00000000-0000-0000-0000-000000000001", 100, "Note"))
		repo.addItem(makeItem("00000000-0000-0000-0000-000000000002", 200, "Note"))
		svc := NewService(repo)

		resp, err := svc.Sync(context.Background(), SyncRequest{
			UserUUID:             userUUID,
			ComputeIntegrityHash: true,
		})
		if err != nil {
			t.Fatalf("Sync() error = %v", err)
		}
		if resp.IntegrityHash == nil || *resp.IntegrityHash == "" {
			t.Error("integrity hash is empty")
		}
	})

	t.Run("content_type_filter", func(t *testing.T) {
		t.Parallel()
		repo := newFakeItemRepo()
		repo.addItem(makeItem("00000000-0000-0000-0000-000000000001", 100, "Note"))
		repo.addItem(makeItem("00000000-0000-0000-0000-000000000002", 200, "Tag"))
		svc := NewService(repo)

		resp, err := svc.Sync(context.Background(), SyncRequest{
			UserUUID:    userUUID,
			ContentType: "Note",
		})
		if err != nil {
			t.Fatalf("Sync() error = %v", err)
		}
		if len(resp.RetrievedItems) != 1 {
			t.Errorf("retrieved %d items, want 1", len(resp.RetrievedItems))
		}
	})

	t.Run("first_sync_frontloads_priority_items", func(t *testing.T) {
		t.Parallel()
		repo := newFakeItemRepo()
		repo.addItem(makeItem("00000000-0000-0000-0000-000000000001", 100, "Note"))
		repo.addItem(makeItem("00000000-0000-0000-0000-000000000002", 200, "SN|ItemsKey"))
		repo.addItem(makeItem("00000000-0000-0000-0000-000000000003", 300, "SN|UserPrefs"))
		svc := NewService(repo)

		resp, err := svc.Sync(context.Background(), SyncRequest{UserUUID: userUUID})
		if err != nil {
			t.Fatalf("Sync() error = %v", err)
		}
		if len(resp.RetrievedItems) < 3 {
			t.Fatalf("retrieved %d items, want 3", len(resp.RetrievedItems))
		}
		if resp.RetrievedItems[0].ContentType != "SN|ItemsKey" {
			t.Errorf("first item type = %q, want SN|ItemsKey", resp.RetrievedItems[0].ContentType)
		}
		if resp.RetrievedItems[1].ContentType != "SN|UserPrefs" {
			t.Errorf("second item type = %q, want SN|UserPrefs", resp.RetrievedItems[1].ContentType)
		}
	})
}
