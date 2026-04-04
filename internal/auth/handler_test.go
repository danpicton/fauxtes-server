package auth

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func newTestHandler() (*Handler, *fakeUserRepo, *fakeSessionRepo) {
	svc, users, sessions := newTestService()
	h := NewHandler(svc)
	return h, users, sessions
}

func TestHandlerRegister(t *testing.T) {
	t.Parallel()

	t.Run("200_success", func(t *testing.T) {
		t.Parallel()
		h, _, _ := newTestHandler()

		body := `{"email":"test@example.com","password":"pass123","api":"004","pw_nonce":"nonce","version":"004"}`
		req := httptest.NewRequest("POST", "/auth", bytes.NewBufferString(body))
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()

		h.Register(rec, req)

		if rec.Code != http.StatusOK {
			t.Errorf("status = %d, want %d; body: %s", rec.Code, http.StatusOK, rec.Body.String())
		}
		var resp AuthResponse
		if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
			t.Fatalf("decode response: %v", err)
		}
		if resp.User.Email != "test@example.com" {
			t.Errorf("email = %q, want %q", resp.User.Email, "test@example.com")
		}
		if resp.Session.AccessToken == "" {
			t.Error("access token is empty")
		}
	})

	t.Run("400_invalid_body", func(t *testing.T) {
		t.Parallel()
		h, _, _ := newTestHandler()

		req := httptest.NewRequest("POST", "/auth", bytes.NewBufferString("not json"))
		rec := httptest.NewRecorder()

		h.Register(rec, req)

		if rec.Code != http.StatusBadRequest {
			t.Errorf("status = %d, want %d", rec.Code, http.StatusBadRequest)
		}
	})

	t.Run("409_duplicate_email", func(t *testing.T) {
		t.Parallel()
		h, _, _ := newTestHandler()

		body := `{"email":"dup@example.com","password":"pass","api":"004","version":"004"}`
		req1 := httptest.NewRequest("POST", "/auth", bytes.NewBufferString(body))
		rec1 := httptest.NewRecorder()
		h.Register(rec1, req1)

		req2 := httptest.NewRequest("POST", "/auth", bytes.NewBufferString(body))
		rec2 := httptest.NewRecorder()
		h.Register(rec2, req2)

		if rec2.Code != http.StatusConflict {
			t.Errorf("status = %d, want %d", rec2.Code, http.StatusConflict)
		}
	})
}

func TestHandlerGetParams(t *testing.T) {
	t.Parallel()

	t.Run("200_existing_user", func(t *testing.T) {
		t.Parallel()
		h, _, _ := newTestHandler()

		// Register first
		regBody := `{"email":"user@example.com","password":"pass","api":"004","pw_nonce":"mynonce","version":"004"}`
		regReq := httptest.NewRequest("POST", "/auth", bytes.NewBufferString(regBody))
		regRec := httptest.NewRecorder()
		h.Register(regRec, regReq)

		req := httptest.NewRequest("GET", "/auth/params?email=user@example.com", nil)
		rec := httptest.NewRecorder()
		h.GetParams(rec, req)

		if rec.Code != http.StatusOK {
			t.Errorf("status = %d, want %d", rec.Code, http.StatusOK)
		}
		var kp map[string]any
		json.NewDecoder(rec.Body).Decode(&kp)
		if kp["identifier"] != "user@example.com" {
			t.Errorf("identifier = %v, want %q", kp["identifier"], "user@example.com")
		}
	})

	t.Run("200_nonexistent_user_pseudo_params", func(t *testing.T) {
		t.Parallel()
		h, _, _ := newTestHandler()

		req := httptest.NewRequest("GET", "/auth/params?email=nobody@example.com", nil)
		rec := httptest.NewRecorder()
		h.GetParams(rec, req)

		if rec.Code != http.StatusOK {
			t.Errorf("status = %d, want %d", rec.Code, http.StatusOK)
		}
		var kp map[string]any
		json.NewDecoder(rec.Body).Decode(&kp)
		if kp["identifier"] != "nobody@example.com" {
			t.Errorf("identifier = %v, want %q", kp["identifier"], "nobody@example.com")
		}
		if kp["pw_nonce"] == nil || kp["pw_nonce"] == "" {
			t.Error("pw_nonce is empty for pseudo params")
		}
	})

	t.Run("400_missing_email", func(t *testing.T) {
		t.Parallel()
		h, _, _ := newTestHandler()

		req := httptest.NewRequest("GET", "/auth/params", nil)
		rec := httptest.NewRecorder()
		h.GetParams(rec, req)

		if rec.Code != http.StatusBadRequest {
			t.Errorf("status = %d, want %d", rec.Code, http.StatusBadRequest)
		}
	})
}

func TestHandlerSignIn(t *testing.T) {
	t.Parallel()

	registerUser := func(t *testing.T, h *Handler) {
		t.Helper()
		body := `{"email":"user@example.com","password":"correct-pass","api":"004","pw_nonce":"n","version":"004"}`
		req := httptest.NewRequest("POST", "/auth", bytes.NewBufferString(body))
		rec := httptest.NewRecorder()
		h.Register(rec, req)
		if rec.Code != http.StatusOK {
			t.Fatalf("setup register failed: %d", rec.Code)
		}
	}

	t.Run("200_valid_credentials", func(t *testing.T) {
		t.Parallel()
		h, _, _ := newTestHandler()
		registerUser(t, h)

		body := `{"email":"user@example.com","password":"correct-pass","api":"004","code_verifier":"v"}`
		req := httptest.NewRequest("POST", "/auth/sign_in", bytes.NewBufferString(body))
		rec := httptest.NewRecorder()
		h.SignIn(rec, req)

		if rec.Code != http.StatusOK {
			t.Errorf("status = %d, want %d; body: %s", rec.Code, http.StatusOK, rec.Body.String())
		}
	})

	t.Run("401_invalid_credentials", func(t *testing.T) {
		t.Parallel()
		h, _, _ := newTestHandler()
		registerUser(t, h)

		body := `{"email":"user@example.com","password":"wrong","api":"004","code_verifier":"v"}`
		req := httptest.NewRequest("POST", "/auth/sign_in", bytes.NewBufferString(body))
		rec := httptest.NewRecorder()
		h.SignIn(rec, req)

		if rec.Code != http.StatusUnauthorized {
			t.Errorf("status = %d, want %d", rec.Code, http.StatusUnauthorized)
		}
	})

	t.Run("410_missing_code_verifier", func(t *testing.T) {
		t.Parallel()
		h, _, _ := newTestHandler()
		registerUser(t, h)

		body := `{"email":"user@example.com","password":"correct-pass","api":"004"}`
		req := httptest.NewRequest("POST", "/auth/sign_in", bytes.NewBufferString(body))
		rec := httptest.NewRecorder()
		h.SignIn(rec, req)

		if rec.Code != http.StatusGone {
			t.Errorf("status = %d, want %d", rec.Code, http.StatusGone)
		}
	})
}

func TestHandlerRefreshSession(t *testing.T) {
	t.Parallel()

	t.Run("200_valid_refresh", func(t *testing.T) {
		t.Parallel()
		h, _, _ := newTestHandler()

		// Register to get session tokens
		regBody := `{"email":"user@example.com","password":"pass","api":"004","version":"004"}`
		regReq := httptest.NewRequest("POST", "/auth", bytes.NewBufferString(regBody))
		regRec := httptest.NewRecorder()
		h.Register(regRec, regReq)

		var regResp AuthResponse
		json.NewDecoder(regRec.Body).Decode(&regResp)

		refreshBody, _ := json.Marshal(RefreshSessionRequest{
			AccessToken:  regResp.Session.AccessToken,
			RefreshToken: regResp.Session.RefreshToken,
		})
		req := httptest.NewRequest("POST", "/session/token", bytes.NewBuffer(refreshBody))
		rec := httptest.NewRecorder()
		h.RefreshSession(rec, req)

		if rec.Code != http.StatusOK {
			t.Errorf("status = %d, want %d; body: %s", rec.Code, http.StatusOK, rec.Body.String())
		}
	})

	t.Run("401_invalid_refresh", func(t *testing.T) {
		t.Parallel()
		h, _, _ := newTestHandler()

		body := `{"access_token":"bogus","refresh_token":"bogus"}`
		req := httptest.NewRequest("POST", "/session/token", bytes.NewBufferString(body))
		rec := httptest.NewRecorder()
		h.RefreshSession(rec, req)

		if rec.Code != http.StatusUnauthorized {
			t.Errorf("status = %d, want %d", rec.Code, http.StatusUnauthorized)
		}
	})
}

func TestHandlerSignOut(t *testing.T) {
	t.Parallel()

	t.Run("204_success", func(t *testing.T) {
		t.Parallel()
		h, _, sessions := newTestHandler()

		// Register to create a session
		regBody := `{"email":"user@example.com","password":"pass","api":"004","version":"004"}`
		regReq := httptest.NewRequest("POST", "/auth", bytes.NewBufferString(regBody))
		regRec := httptest.NewRecorder()
		h.Register(regRec, regReq)

		// Get session UUID
		var sessionUUID string
		for id := range sessions.sessions {
			sessionUUID = id
			break
		}

		// Sign out with session context set (simulating middleware)
		req := httptest.NewRequest("DELETE", "/session", nil)
		ctx := req.Context()
		ctx = contextWithSession(ctx, sessionUUID)
		req = req.WithContext(ctx)
		rec := httptest.NewRecorder()

		h.SignOut(rec, req)

		if rec.Code != http.StatusNoContent {
			t.Errorf("status = %d, want %d; body: %s", rec.Code, http.StatusNoContent, rec.Body.String())
		}
	})
}

func contextWithSession(ctx context.Context, sessionUUID string) context.Context {
	ctx = context.WithValue(ctx, ContextKeySessionUUID, sessionUUID)
	ctx = context.WithValue(ctx, ContextKeyUserUUID, "user-uuid")
	return ctx
}
