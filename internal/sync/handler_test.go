package sync

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/danpicton/fauxtes-server/internal/auth"
	"github.com/danpicton/fauxtes-server/internal/domain"
)

func TestHandlerSync(t *testing.T) {
	t.Parallel()

	userUUID := "a4b8e3c2-1f6d-4a3b-9c7e-8d2f5a6b3c4d"
	sessionUUID := "b5c9f4d3-2e7a-4b8c-ad6f-9e3a7b4c5d6e"

	withAuth := func(r *http.Request) *http.Request {
		ctx := r.Context()
		ctx = context.WithValue(ctx, auth.ContextKeyUserUUID, userUUID)
		ctx = context.WithValue(ctx, auth.ContextKeySessionUUID, sessionUUID)
		ctx = context.WithValue(ctx, auth.ContextKeyReadOnly, false)
		return r.WithContext(ctx)
	}

	t.Run("200_basic_sync", func(t *testing.T) {
		t.Parallel()
		repo := newFakeItemRepo()
		svc := NewService(repo)
		h := NewHandler(svc)

		body := `{"items":[]}`
		req := withAuth(httptest.NewRequest("POST", "/items/sync", bytes.NewBufferString(body)))
		rec := httptest.NewRecorder()
		h.Sync(rec, req)

		if rec.Code != http.StatusOK {
			t.Errorf("status = %d, want %d; body: %s", rec.Code, http.StatusOK, rec.Body.String())
		}
	})

	t.Run("200_with_items", func(t *testing.T) {
		t.Parallel()
		repo := newFakeItemRepo()
		svc := NewService(repo)
		h := NewHandler(svc)

		body := `{"items":[{"uuid":"00000000-0000-0000-0000-000000000001","content":"enc","content_type":"Note"}]}`
		req := withAuth(httptest.NewRequest("POST", "/items/sync", bytes.NewBufferString(body)))
		rec := httptest.NewRecorder()
		h.Sync(rec, req)

		if rec.Code != http.StatusOK {
			t.Errorf("status = %d, want %d", rec.Code, http.StatusOK)
		}
		var resp SyncResponse
		json.NewDecoder(rec.Body).Decode(&resp)
		if len(resp.SavedItems) != 1 {
			t.Errorf("saved %d items, want 1", len(resp.SavedItems))
		}
	})

	t.Run("200_with_sync_token", func(t *testing.T) {
		t.Parallel()
		repo := newFakeItemRepo()
		repo.addItem(&domain.Item{
			UUID: "00000000-0000-0000-0000-000000000001", UserUUID: userUUID,
			Content: strPtr("enc"), ContentType: strPtr("Note"),
			UpdatedAtTimestamp: 1000, CreatedAtTimestamp: 1000,
			CreatedAt: time.Now(), UpdatedAt: time.Now(),
		})
		svc := NewService(repo)
		h := NewHandler(svc)

		syncToken := domain.EncodeSyncToken(500)
		body, _ := json.Marshal(SyncRequest{SyncToken: syncToken})
		req := withAuth(httptest.NewRequest("POST", "/items/sync", bytes.NewBuffer(body)))
		rec := httptest.NewRecorder()
		h.Sync(rec, req)

		if rec.Code != http.StatusOK {
			t.Errorf("status = %d, want %d", rec.Code, http.StatusOK)
		}
		var resp SyncResponse
		json.NewDecoder(rec.Body).Decode(&resp)
		if len(resp.RetrievedItems) != 1 {
			t.Errorf("retrieved %d items, want 1", len(resp.RetrievedItems))
		}
	})

	t.Run("401_unauthenticated", func(t *testing.T) {
		t.Parallel()
		repo := newFakeItemRepo()
		svc := NewService(repo)
		h := NewHandler(svc)

		req := httptest.NewRequest("POST", "/items/sync", bytes.NewBufferString(`{}`))
		rec := httptest.NewRecorder()
		h.Sync(rec, req)

		if rec.Code != http.StatusUnauthorized {
			t.Errorf("status = %d, want %d", rec.Code, http.StatusUnauthorized)
		}
	})

	t.Run("400_invalid_body", func(t *testing.T) {
		t.Parallel()
		repo := newFakeItemRepo()
		svc := NewService(repo)
		h := NewHandler(svc)

		req := withAuth(httptest.NewRequest("POST", "/items/sync", bytes.NewBufferString("not json")))
		rec := httptest.NewRecorder()
		h.Sync(rec, req)

		if rec.Code != http.StatusBadRequest {
			t.Errorf("status = %d, want %d", rec.Code, http.StatusBadRequest)
		}
	})

	t.Run("200_first_sync_empty", func(t *testing.T) {
		t.Parallel()
		repo := newFakeItemRepo()
		svc := NewService(repo)
		h := NewHandler(svc)

		body := `{}`
		req := withAuth(httptest.NewRequest("POST", "/items/sync", bytes.NewBufferString(body)))
		rec := httptest.NewRecorder()
		h.Sync(rec, req)

		if rec.Code != http.StatusOK {
			t.Errorf("status = %d, want %d", rec.Code, http.StatusOK)
		}
		var resp SyncResponse
		json.NewDecoder(rec.Body).Decode(&resp)
		if len(resp.RetrievedItems) != 0 {
			t.Errorf("retrieved %d items, want 0", len(resp.RetrievedItems))
		}
	})
}
