package auth

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/danpicton/fauxtes-server/internal/domain"
)

func TestAuthMiddleware(t *testing.T) {
	t.Parallel()

	newSessionRepo := func(accessToken string, expiration time.Time, readonly bool) *fakeSessionRepo {
		t.Helper()
		repo := newFakeSessionRepo()
		s := domain.NewSession("user-uuid-123")
		s.HashedAccessToken = HashToken(accessToken)
		s.HashedRefreshToken = HashToken("refresh-token")
		s.AccessExpiration = expiration
		s.ReadonlyAccess = readonly
		repo.sessions[s.UUID] = &s
		repo.byAccessToken[s.HashedAccessToken] = &s
		return repo
	}

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		userUUID := UserUUIDFromContext(r.Context())
		w.Write([]byte(userUUID))
	})

	t.Run("valid_token_sets_user_context", func(t *testing.T) {
		t.Parallel()
		repo := newSessionRepo("valid-token", time.Now().Add(1*time.Hour), false)
		mw := AuthMiddleware(repo)(handler)

		req := httptest.NewRequest("GET", "/test", nil)
		req.Header.Set("Authorization", "Bearer valid-token")
		rec := httptest.NewRecorder()

		mw.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Errorf("status = %d, want %d", rec.Code, http.StatusOK)
		}
		if rec.Body.String() != "user-uuid-123" {
			t.Errorf("body = %q, want %q", rec.Body.String(), "user-uuid-123")
		}
	})

	t.Run("missing_header_returns_401", func(t *testing.T) {
		t.Parallel()
		repo := newFakeSessionRepo()
		mw := AuthMiddleware(repo)(handler)

		req := httptest.NewRequest("GET", "/test", nil)
		rec := httptest.NewRecorder()

		mw.ServeHTTP(rec, req)

		if rec.Code != http.StatusUnauthorized {
			t.Errorf("status = %d, want %d", rec.Code, http.StatusUnauthorized)
		}
	})

	t.Run("invalid_token_returns_401", func(t *testing.T) {
		t.Parallel()
		repo := newSessionRepo("valid-token", time.Now().Add(1*time.Hour), false)
		mw := AuthMiddleware(repo)(handler)

		req := httptest.NewRequest("GET", "/test", nil)
		req.Header.Set("Authorization", "Bearer wrong-token")
		rec := httptest.NewRecorder()

		mw.ServeHTTP(rec, req)

		if rec.Code != http.StatusUnauthorized {
			t.Errorf("status = %d, want %d", rec.Code, http.StatusUnauthorized)
		}
	})

	t.Run("expired_token_returns_401", func(t *testing.T) {
		t.Parallel()
		repo := newSessionRepo("expired-token", time.Now().Add(-1*time.Hour), false)
		mw := AuthMiddleware(repo)(handler)

		req := httptest.NewRequest("GET", "/test", nil)
		req.Header.Set("Authorization", "Bearer expired-token")
		rec := httptest.NewRecorder()

		mw.ServeHTTP(rec, req)

		if rec.Code != http.StatusUnauthorized {
			t.Errorf("status = %d, want %d", rec.Code, http.StatusUnauthorized)
		}
	})

	t.Run("readonly_flag_set", func(t *testing.T) {
		t.Parallel()
		repo := newSessionRepo("ro-token", time.Now().Add(1*time.Hour), true)

		readonlyHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if ReadOnlyFromContext(r.Context()) {
				w.Write([]byte("readonly"))
			} else {
				w.Write([]byte("readwrite"))
			}
		})
		mw := AuthMiddleware(repo)(readonlyHandler)

		req := httptest.NewRequest("GET", "/test", nil)
		req.Header.Set("Authorization", "Bearer ro-token")
		rec := httptest.NewRecorder()

		mw.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Errorf("status = %d, want %d", rec.Code, http.StatusOK)
		}
		if rec.Body.String() != "readonly" {
			t.Errorf("body = %q, want %q", rec.Body.String(), "readonly")
		}
	})
}
