package sqlite

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/danpicton/fauxtes-server/internal/domain"
	"github.com/danpicton/fauxtes-server/internal/sync"
)

func testDB(t *testing.T) *testDBHelper {
	t.Helper()
	db, err := OpenInMemory()
	if err != nil {
		t.Fatalf("OpenInMemory() error = %v", err)
	}
	t.Cleanup(func() { db.Close() })
	return &testDBHelper{
		UserRepo:    NewUserRepo(db),
		SessionRepo: NewSessionRepo(db),
		ItemRepo:    NewItemRepo(db),
	}
}

type testDBHelper struct {
	*UserRepo
	*SessionRepo
	*ItemRepo
}

// --- User Repo Tests ---

func TestUserRepo(t *testing.T) {
	t.Run("create_and_find_by_email", func(t *testing.T) {
		h := testDB(t)
		ctx := context.Background()

		u := domain.NewUser("test@example.com")
		u.EncryptedPassword = "hashed"
		nonce := "test-nonce"
		u.PwNonce = &nonce

		if err := h.UserRepo.Create(ctx, &u); err != nil {
			t.Fatalf("Create() error = %v", err)
		}

		found, err := h.UserRepo.FindByEmail(ctx, "test@example.com")
		if err != nil {
			t.Fatalf("FindByEmail() error = %v", err)
		}
		if found == nil {
			t.Fatal("FindByEmail() returned nil")
		}
		if found.UUID != u.UUID {
			t.Errorf("UUID = %q, want %q", found.UUID, u.UUID)
		}
		if found.Email != "test@example.com" {
			t.Errorf("Email = %q, want %q", found.Email, "test@example.com")
		}
	})

	t.Run("find_by_email_not_found", func(t *testing.T) {
		h := testDB(t)
		found, err := h.UserRepo.FindByEmail(context.Background(), "nobody@example.com")
		if err != nil {
			t.Fatalf("FindByEmail() error = %v", err)
		}
		if found != nil {
			t.Error("expected nil for non-existent user")
		}
	})

	t.Run("duplicate_email_returns_error", func(t *testing.T) {
		h := testDB(t)
		ctx := context.Background()

		u1 := domain.NewUser("dup@example.com")
		u1.EncryptedPassword = "hash1"
		u2 := domain.NewUser("dup@example.com")
		u2.EncryptedPassword = "hash2"

		_ = h.UserRepo.Create(ctx, &u1)
		err := h.UserRepo.Create(ctx, &u2)
		if err == nil {
			t.Error("expected error for duplicate email")
		}
	})
}

// --- Session Repo Tests ---

func TestSessionRepo(t *testing.T) {
	t.Run("create_and_find_by_access_token", func(t *testing.T) {
		h := testDB(t)
		ctx := context.Background()

		// Create a user first (FK constraint)
		u := domain.NewUser("test@example.com")
		u.EncryptedPassword = "hash"
		_ = h.UserRepo.Create(ctx, &u)

		s := domain.NewSession(u.UUID)
		s.HashedAccessToken = "hashed-access"
		s.HashedRefreshToken = "hashed-refresh"
		s.AccessExpiration = time.Now().Add(1 * time.Hour)
		s.RefreshExpiration = time.Now().Add(24 * time.Hour)

		if err := h.SessionRepo.Create(ctx, &s); err != nil {
			t.Fatalf("Create() error = %v", err)
		}

		found, err := h.SessionRepo.FindByAccessToken(ctx, "hashed-access")
		if err != nil {
			t.Fatalf("FindByAccessToken() error = %v", err)
		}
		if found == nil {
			t.Fatal("FindByAccessToken() returned nil")
		}
		if found.UUID != s.UUID {
			t.Errorf("UUID = %q, want %q", found.UUID, s.UUID)
		}
	})

	t.Run("delete", func(t *testing.T) {
		h := testDB(t)
		ctx := context.Background()

		u := domain.NewUser("test@example.com")
		u.EncryptedPassword = "hash"
		_ = h.UserRepo.Create(ctx, &u)

		s := domain.NewSession(u.UUID)
		s.HashedAccessToken = "access"
		s.HashedRefreshToken = "refresh"
		s.AccessExpiration = time.Now().Add(1 * time.Hour)
		s.RefreshExpiration = time.Now().Add(24 * time.Hour)
		_ = h.SessionRepo.Create(ctx, &s)

		if err := h.SessionRepo.Delete(ctx, s.UUID); err != nil {
			t.Fatalf("Delete() error = %v", err)
		}

		found, _ := h.SessionRepo.FindByUUID(ctx, s.UUID)
		if found != nil {
			t.Error("session not deleted")
		}
	})

	t.Run("delete_all_for_user", func(t *testing.T) {
		h := testDB(t)
		ctx := context.Background()

		u := domain.NewUser("test@example.com")
		u.EncryptedPassword = "hash"
		_ = h.UserRepo.Create(ctx, &u)

		for i := 0; i < 3; i++ {
			s := domain.NewSession(u.UUID)
			s.HashedAccessToken = "access-" + string(rune('a'+i))
			s.HashedRefreshToken = "refresh-" + string(rune('a'+i))
			s.AccessExpiration = time.Now().Add(1 * time.Hour)
			s.RefreshExpiration = time.Now().Add(24 * time.Hour)
			_ = h.SessionRepo.Create(ctx, &s)
		}

		if err := h.SessionRepo.DeleteAllForUser(ctx, u.UUID); err != nil {
			t.Fatalf("DeleteAllForUser() error = %v", err)
		}
	})
}

// --- Item Repo Tests ---

func TestItemRepo(t *testing.T) {
	userUUID := "a4b8e3c2-1f6d-4a3b-9c7e-8d2f5a6b3c4d"

	setupUser := func(t *testing.T, h *testDBHelper) {
		t.Helper()
		u := &domain.User{
			UUID: userUUID, Email: "test@example.com", EncryptedPassword: "hash",
			CreatedAt: time.Now().UTC(), UpdatedAt: time.Now().UTC(),
		}
		_ = h.UserRepo.Create(context.Background(), u)
	}

	t.Run("save_new_and_find", func(t *testing.T) {
		h := testDB(t)
		setupUser(t, h)
		ctx := context.Background()

		ct := "Note"
		content := "encrypted-content"
		item := &domain.Item{
			UUID: "00000000-0000-0000-0000-000000000001", UserUUID: userUUID,
			Content: &content, ContentType: &ct,
			CreatedAt: time.Now().UTC(), UpdatedAt: time.Now().UTC(),
			CreatedAtTimestamp: 1000, UpdatedAtTimestamp: 1000,
		}

		if err := h.ItemRepo.Save(ctx, item); err != nil {
			t.Fatalf("Save() error = %v", err)
		}

		found, err := h.ItemRepo.FindByUUID(ctx, item.UUID)
		if err != nil {
			t.Fatalf("FindByUUID() error = %v", err)
		}
		if found == nil {
			t.Fatal("FindByUUID() returned nil")
		}
		if found.Content == nil || *found.Content != "encrypted-content" {
			t.Errorf("Content = %v, want %q", found.Content, "encrypted-content")
		}
	})

	t.Run("save_existing_updates", func(t *testing.T) {
		h := testDB(t)
		setupUser(t, h)
		ctx := context.Background()

		ct := "Note"
		c1 := "original"
		item := &domain.Item{
			UUID: "00000000-0000-0000-0000-000000000001", UserUUID: userUUID,
			Content: &c1, ContentType: &ct,
			CreatedAt: time.Now().UTC(), UpdatedAt: time.Now().UTC(),
			CreatedAtTimestamp: 1000, UpdatedAtTimestamp: 1000,
		}
		_ = h.ItemRepo.Save(ctx, item)

		c2 := "updated"
		item.Content = &c2
		item.UpdatedAtTimestamp = 2000
		_ = h.ItemRepo.Save(ctx, item)

		found, _ := h.ItemRepo.FindByUUID(ctx, item.UUID)
		if found.Content == nil || *found.Content != "updated" {
			t.Errorf("Content = %v, want %q", found.Content, "updated")
		}
	})

	t.Run("find_by_user_since_timestamp", func(t *testing.T) {
		h := testDB(t)
		setupUser(t, h)
		ctx := context.Background()

		ct := "Note"
		for i, ts := range []int64{100, 200, 300} {
			c := "content"
			item := &domain.Item{
				UUID: fakeUUID(i), UserUUID: userUUID,
				Content: &c, ContentType: &ct,
				CreatedAt: time.Now().UTC(), UpdatedAt: time.Now().UTC(),
				CreatedAtTimestamp: ts, UpdatedAtTimestamp: ts,
			}
			_ = h.ItemRepo.Save(ctx, item)
		}

		items, err := h.ItemRepo.FindByUserUUID(ctx, userUUID, sync.FindItemsOpts{SinceTimestamp: 150})
		if err != nil {
			t.Fatalf("FindByUserUUID() error = %v", err)
		}
		if len(items) != 2 {
			t.Errorf("got %d items, want 2", len(items))
		}
	})

	t.Run("find_by_user_with_limit", func(t *testing.T) {
		h := testDB(t)
		setupUser(t, h)
		ctx := context.Background()

		ct := "Note"
		for i := 0; i < 10; i++ {
			c := "content"
			item := &domain.Item{
				UUID: fakeUUID(i), UserUUID: userUUID,
				Content: &c, ContentType: &ct,
				CreatedAt: time.Now().UTC(), UpdatedAt: time.Now().UTC(),
				CreatedAtTimestamp: int64(i * 100), UpdatedAtTimestamp: int64(i * 100),
			}
			_ = h.ItemRepo.Save(ctx, item)
		}

		items, err := h.ItemRepo.FindByUserUUID(ctx, userUUID, sync.FindItemsOpts{Limit: 5})
		if err != nil {
			t.Fatalf("FindByUserUUID() error = %v", err)
		}
		if len(items) != 5 {
			t.Errorf("got %d items, want 5", len(items))
		}
	})

	t.Run("find_by_user_content_type_filter", func(t *testing.T) {
		h := testDB(t)
		setupUser(t, h)
		ctx := context.Background()

		note := "Note"
		tag := "Tag"
		c := "content"
		_ = h.ItemRepo.Save(ctx, &domain.Item{
			UUID: "00000000-0000-0000-0000-000000000001", UserUUID: userUUID,
			Content: &c, ContentType: &note,
			CreatedAt: time.Now().UTC(), UpdatedAt: time.Now().UTC(),
			CreatedAtTimestamp: 100, UpdatedAtTimestamp: 100,
		})
		_ = h.ItemRepo.Save(ctx, &domain.Item{
			UUID: "00000000-0000-0000-0000-000000000002", UserUUID: userUUID,
			Content: &c, ContentType: &tag,
			CreatedAt: time.Now().UTC(), UpdatedAt: time.Now().UTC(),
			CreatedAtTimestamp: 200, UpdatedAtTimestamp: 200,
		})

		items, err := h.ItemRepo.FindByUserUUID(ctx, userUUID, sync.FindItemsOpts{ContentType: "Note"})
		if err != nil {
			t.Fatalf("FindByUserUUID() error = %v", err)
		}
		if len(items) != 1 {
			t.Errorf("got %d items, want 1", len(items))
		}
	})

	t.Run("sort_by_timestamp_asc", func(t *testing.T) {
		h := testDB(t)
		setupUser(t, h)
		ctx := context.Background()

		ct := "Note"
		c := "content"
		// Insert in reverse order
		for i, ts := range []int64{300, 100, 200} {
			_ = h.ItemRepo.Save(ctx, &domain.Item{
				UUID: fakeUUID(i), UserUUID: userUUID,
				Content: &c, ContentType: &ct,
				CreatedAt: time.Now().UTC(), UpdatedAt: time.Now().UTC(),
				CreatedAtTimestamp: ts, UpdatedAtTimestamp: ts,
			})
		}

		items, _ := h.ItemRepo.FindByUserUUID(ctx, userUUID, sync.FindItemsOpts{})
		if len(items) != 3 {
			t.Fatalf("got %d items, want 3", len(items))
		}
		if items[0].UpdatedAtTimestamp > items[1].UpdatedAtTimestamp ||
			items[1].UpdatedAtTimestamp > items[2].UpdatedAtTimestamp {
			t.Error("items not sorted by timestamp ASC")
		}
	})
}

func fakeUUID(n int) string {
	return fmt.Sprintf("00000000-0000-0000-0000-%012d", n)
}
