package auth

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/danpicton/fauxtes-server/internal/domain"
)

// fakeUserRepo is an in-memory UserRepository for testing.
type fakeUserRepo struct {
	users     map[string]*domain.User // keyed by UUID
	byEmail   map[string]*domain.User
	createErr error
}

func newFakeUserRepo() *fakeUserRepo {
	return &fakeUserRepo{
		users:   make(map[string]*domain.User),
		byEmail: make(map[string]*domain.User),
	}
}

func (r *fakeUserRepo) Create(_ context.Context, u *domain.User) error {
	if r.createErr != nil {
		return r.createErr
	}
	r.users[u.UUID] = u
	r.byEmail[u.Email] = u
	return nil
}

func (r *fakeUserRepo) FindByEmail(_ context.Context, email string) (*domain.User, error) {
	return r.byEmail[email], nil
}

func (r *fakeUserRepo) FindByUUID(_ context.Context, uuid string) (*domain.User, error) {
	return r.users[uuid], nil
}

func (r *fakeUserRepo) Update(_ context.Context, u *domain.User) error {
	r.users[u.UUID] = u
	r.byEmail[u.Email] = u
	return nil
}

// fakeSessionRepo is an in-memory SessionRepository for testing.
type fakeSessionRepo struct {
	sessions       map[string]*domain.Session // keyed by UUID
	byAccessToken  map[string]*domain.Session
	byRefreshToken map[string]*domain.Session
}

func newFakeSessionRepo() *fakeSessionRepo {
	return &fakeSessionRepo{
		sessions:       make(map[string]*domain.Session),
		byAccessToken:  make(map[string]*domain.Session),
		byRefreshToken: make(map[string]*domain.Session),
	}
}

func (r *fakeSessionRepo) Create(_ context.Context, s *domain.Session) error {
	cp := *s
	r.sessions[s.UUID] = &cp
	r.byAccessToken[s.HashedAccessToken] = &cp
	r.byRefreshToken[s.HashedRefreshToken] = &cp
	return nil
}

func (r *fakeSessionRepo) FindByUUID(_ context.Context, uuid string) (*domain.Session, error) {
	s := r.sessions[uuid]
	if s == nil {
		return nil, nil
	}
	cp := *s
	return &cp, nil
}

func (r *fakeSessionRepo) FindByAccessToken(_ context.Context, hashedToken string) (*domain.Session, error) {
	s := r.byAccessToken[hashedToken]
	if s == nil {
		return nil, nil
	}
	cp := *s
	return &cp, nil
}

func (r *fakeSessionRepo) FindByRefreshToken(_ context.Context, hashedToken string) (*domain.Session, error) {
	s := r.byRefreshToken[hashedToken]
	if s == nil {
		return nil, nil
	}
	cp := *s
	return &cp, nil
}

func (r *fakeSessionRepo) Delete(_ context.Context, uuid string) error {
	s, ok := r.sessions[uuid]
	if !ok {
		return nil
	}
	delete(r.byAccessToken, s.HashedAccessToken)
	delete(r.byRefreshToken, s.HashedRefreshToken)
	delete(r.sessions, uuid)
	return nil
}

func (r *fakeSessionRepo) DeleteAllForUser(_ context.Context, userUUID string) error {
	for id, s := range r.sessions {
		if s.UserUUID == userUUID {
			delete(r.byAccessToken, s.HashedAccessToken)
			delete(r.byRefreshToken, s.HashedRefreshToken)
			delete(r.sessions, id)
		}
	}
	return nil
}

func (r *fakeSessionRepo) Update(_ context.Context, s *domain.Session) error {
	// Remove old token mappings
	if old, ok := r.sessions[s.UUID]; ok {
		delete(r.byAccessToken, old.HashedAccessToken)
		delete(r.byRefreshToken, old.HashedRefreshToken)
	}
	cp := *s
	r.sessions[s.UUID] = &cp
	r.byAccessToken[s.HashedAccessToken] = &cp
	r.byRefreshToken[s.HashedRefreshToken] = &cp
	return nil
}

func newTestService() (*Service, *fakeUserRepo, *fakeSessionRepo) {
	users := newFakeUserRepo()
	sessions := newFakeSessionRepo()
	svc := NewService(users, sessions, DefaultServiceConfig())
	return svc, users, sessions
}

// --- Register Tests ---

func TestRegister(t *testing.T) {
	t.Parallel()

	t.Run("success_creates_user_and_session", func(t *testing.T) {
		t.Parallel()
		svc, users, sessions := newTestService()
		resp, err := svc.Register(context.Background(), RegisterRequest{
			Email:    "test@example.com",
			Password: "password123",
			APIVersion: "004",
			PwNonce:  "test-nonce",
			Version:  "004",
		})
		if err != nil {
			t.Fatalf("Register() error = %v", err)
		}
		if resp.User.Email != "test@example.com" {
			t.Errorf("resp.User.Email = %q, want %q", resp.User.Email, "test@example.com")
		}
		if resp.Session.AccessToken == "" {
			t.Error("resp.Session.AccessToken is empty")
		}
		if resp.Session.RefreshToken == "" {
			t.Error("resp.Session.RefreshToken is empty")
		}
		if len(users.users) != 1 {
			t.Errorf("user count = %d, want 1", len(users.users))
		}
		if len(sessions.sessions) != 1 {
			t.Errorf("session count = %d, want 1", len(sessions.sessions))
		}
	})

	t.Run("invalid_email", func(t *testing.T) {
		t.Parallel()
		svc, _, _ := newTestService()
		_, err := svc.Register(context.Background(), RegisterRequest{
			Email:      "",
			Password:   "pass",
			APIVersion: "004",
		})
		if !errors.Is(err, ErrInvalidEmail) {
			t.Errorf("Register() error = %v, want ErrInvalidEmail", err)
		}
	})

	t.Run("duplicate_email", func(t *testing.T) {
		t.Parallel()
		svc, _, _ := newTestService()
		ctx := context.Background()
		_, _ = svc.Register(ctx, RegisterRequest{
			Email: "dup@example.com", Password: "pass", APIVersion: "004", Version: "004",
		})
		_, err := svc.Register(ctx, RegisterRequest{
			Email: "dup@example.com", Password: "pass2", APIVersion: "004", Version: "004",
		})
		if !errors.Is(err, ErrDuplicateEmail) {
			t.Errorf("Register() error = %v, want ErrDuplicateEmail", err)
		}
	})

	t.Run("invalid_api_version", func(t *testing.T) {
		t.Parallel()
		svc, _, _ := newTestService()
		_, err := svc.Register(context.Background(), RegisterRequest{
			Email: "test@example.com", Password: "pass", APIVersion: "999",
		})
		if !errors.Is(err, ErrInvalidAPIVersion) {
			t.Errorf("Register() error = %v, want ErrInvalidAPIVersion", err)
		}
	})

	t.Run("legacy_api_version", func(t *testing.T) {
		t.Parallel()
		svc, _, _ := newTestService()
		_, err := svc.Register(context.Background(), RegisterRequest{
			Email: "test@example.com", Password: "pass", APIVersion: "003",
		})
		if !errors.Is(err, ErrLegacyAPIVersion) {
			t.Errorf("Register() error = %v, want ErrLegacyAPIVersion", err)
		}
	})

	t.Run("disabled_registration", func(t *testing.T) {
		t.Parallel()
		users := newFakeUserRepo()
		sessions := newFakeSessionRepo()
		cfg := DefaultServiceConfig()
		cfg.DisableRegistration = true
		svc := NewService(users, sessions, cfg)

		_, err := svc.Register(context.Background(), RegisterRequest{
			Email: "test@example.com", Password: "pass", APIVersion: "004",
		})
		if !errors.Is(err, ErrRegistrationDisabled) {
			t.Errorf("Register() error = %v, want ErrRegistrationDisabled", err)
		}
	})

	t.Run("stores_password_params", func(t *testing.T) {
		t.Parallel()
		svc, users, _ := newTestService()
		_, err := svc.Register(context.Background(), RegisterRequest{
			Email:      "params@example.com",
			Password:   "pass",
			APIVersion: "004",
			PwNonce:    "my-nonce-value",
			Version:    "004",
		})
		if err != nil {
			t.Fatalf("Register() error = %v", err)
		}
		u := users.byEmail["params@example.com"]
		if u.PwNonce == nil || *u.PwNonce != "my-nonce-value" {
			t.Errorf("stored PwNonce = %v, want %q", u.PwNonce, "my-nonce-value")
		}
	})

	t.Run("generates_encrypted_server_key", func(t *testing.T) {
		t.Parallel()
		svc, users, _ := newTestService()
		_, err := svc.Register(context.Background(), RegisterRequest{
			Email: "key@example.com", Password: "pass", APIVersion: "004", Version: "004",
		})
		if err != nil {
			t.Fatalf("Register() error = %v", err)
		}
		u := users.byEmail["key@example.com"]
		if u.EncryptedServerKey == nil || *u.EncryptedServerKey == "" {
			t.Error("EncryptedServerKey not generated")
		}
		if len(*u.EncryptedServerKey) != 64 {
			t.Errorf("EncryptedServerKey length = %d, want 64", len(*u.EncryptedServerKey))
		}
	})
}

// --- GetParams Tests ---

func TestGetParams(t *testing.T) {
	t.Parallel()

	t.Run("existing_user_returns_v004_params", func(t *testing.T) {
		t.Parallel()
		svc, _, _ := newTestService()
		ctx := context.Background()

		// Register a user first
		_, _ = svc.Register(ctx, RegisterRequest{
			Email: "user@example.com", Password: "pass", APIVersion: "004",
			PwNonce: "test-nonce", Version: "004",
		})

		kp, err := svc.GetParams(ctx, GetParamsRequest{Email: "user@example.com"})
		if err != nil {
			t.Fatalf("GetParams() error = %v", err)
		}
		if kp.Identifier != "user@example.com" {
			t.Errorf("Identifier = %q, want %q", kp.Identifier, "user@example.com")
		}
		if kp.Version != "004" {
			t.Errorf("Version = %q, want %q", kp.Version, "004")
		}
		if kp.PwNonce != "test-nonce" {
			t.Errorf("PwNonce = %q, want %q", kp.PwNonce, "test-nonce")
		}
	})

	t.Run("nonexistent_user_returns_pseudo_params", func(t *testing.T) {
		t.Parallel()
		svc, _, _ := newTestService()
		kp, err := svc.GetParams(context.Background(), GetParamsRequest{Email: "nobody@example.com"})
		if err != nil {
			t.Fatalf("GetParams() error = %v", err)
		}
		if kp.Identifier != "nobody@example.com" {
			t.Errorf("Identifier = %q, want %q", kp.Identifier, "nobody@example.com")
		}
		if kp.PwNonce == "" {
			t.Error("PwNonce is empty for pseudo params")
		}
	})

	t.Run("pseudo_params_are_deterministic", func(t *testing.T) {
		t.Parallel()
		svc, _, _ := newTestService()
		ctx := context.Background()
		kp1, _ := svc.GetParams(ctx, GetParamsRequest{Email: "fake@example.com"})
		kp2, _ := svc.GetParams(ctx, GetParamsRequest{Email: "fake@example.com"})
		if kp1.PwNonce != kp2.PwNonce {
			t.Errorf("pseudo params not deterministic: %q != %q", kp1.PwNonce, kp2.PwNonce)
		}
	})

	t.Run("includes_created_origination_when_set", func(t *testing.T) {
		t.Parallel()
		svc, _, _ := newTestService()
		ctx := context.Background()

		_, _ = svc.Register(ctx, RegisterRequest{
			Email: "orig@example.com", Password: "pass", APIVersion: "004",
			PwNonce: "nonce", Version: "004", Created: "1234567890", Origination: "registration",
		})

		kp, err := svc.GetParams(ctx, GetParamsRequest{Email: "orig@example.com", Authenticated: true})
		if err != nil {
			t.Fatalf("GetParams() error = %v", err)
		}
		if kp.Created != "1234567890" {
			t.Errorf("Created = %q, want %q", kp.Created, "1234567890")
		}
		if kp.Origination != "registration" {
			t.Errorf("Origination = %q, want %q", kp.Origination, "registration")
		}
	})

	t.Run("empty_email_returns_error", func(t *testing.T) {
		t.Parallel()
		svc, _, _ := newTestService()
		_, err := svc.GetParams(context.Background(), GetParamsRequest{Email: ""})
		if !errors.Is(err, ErrInvalidEmail) {
			t.Errorf("GetParams() error = %v, want ErrInvalidEmail", err)
		}
	})
}

// --- SignIn Tests ---

func TestSignIn(t *testing.T) {
	t.Parallel()

	registerUser := func(t *testing.T, svc *Service) {
		t.Helper()
		_, err := svc.Register(context.Background(), RegisterRequest{
			Email: "user@example.com", Password: "correct-password",
			APIVersion: "004", PwNonce: "nonce", Version: "004",
		})
		if err != nil {
			t.Fatalf("setup Register() error = %v", err)
		}
	}

	t.Run("success_with_code_verifier", func(t *testing.T) {
		t.Parallel()
		svc, _, sessions := newTestService()
		registerUser(t, svc)

		resp, err := svc.SignIn(context.Background(), SignInRequest{
			Email: "user@example.com", Password: "correct-password",
			APIVersion: "004", CodeVerifier: "some-verifier",
		})
		if err != nil {
			t.Fatalf("SignIn() error = %v", err)
		}
		if resp.Session.AccessToken == "" {
			t.Error("access token is empty")
		}
		if len(sessions.sessions) != 2 { // 1 from register + 1 from sign-in
			t.Errorf("session count = %d, want 2", len(sessions.sessions))
		}
	})

	t.Run("wrong_password", func(t *testing.T) {
		t.Parallel()
		svc, _, _ := newTestService()
		registerUser(t, svc)

		_, err := svc.SignIn(context.Background(), SignInRequest{
			Email: "user@example.com", Password: "wrong-password",
			APIVersion: "004", CodeVerifier: "verifier",
		})
		if !errors.Is(err, ErrInvalidCredentials) {
			t.Errorf("SignIn() error = %v, want ErrInvalidCredentials", err)
		}
	})

	t.Run("nonexistent_user", func(t *testing.T) {
		t.Parallel()
		svc, _, _ := newTestService()

		_, err := svc.SignIn(context.Background(), SignInRequest{
			Email: "nobody@example.com", Password: "pass",
			APIVersion: "004", CodeVerifier: "verifier",
		})
		if !errors.Is(err, ErrInvalidCredentials) {
			t.Errorf("SignIn() error = %v, want ErrInvalidCredentials", err)
		}
	})

	t.Run("empty_username", func(t *testing.T) {
		t.Parallel()
		svc, _, _ := newTestService()

		_, err := svc.SignIn(context.Background(), SignInRequest{
			Email: "", Password: "pass", APIVersion: "004",
		})
		if !errors.Is(err, ErrInvalidEmail) {
			t.Errorf("SignIn() error = %v, want ErrInvalidEmail", err)
		}
	})

	t.Run("invalid_api_version", func(t *testing.T) {
		t.Parallel()
		svc, _, _ := newTestService()

		_, err := svc.SignIn(context.Background(), SignInRequest{
			Email: "user@example.com", Password: "pass", APIVersion: "999",
		})
		if !errors.Is(err, ErrInvalidAPIVersion) {
			t.Errorf("SignIn() error = %v, want ErrInvalidAPIVersion", err)
		}
	})

	t.Run("v004_requires_code_verifier", func(t *testing.T) {
		t.Parallel()
		svc, _, _ := newTestService()
		registerUser(t, svc)

		_, err := svc.SignIn(context.Background(), SignInRequest{
			Email: "user@example.com", Password: "correct-password",
			APIVersion: "004", CodeVerifier: "",
		})
		if !errors.Is(err, ErrCodeVerifierRequired) {
			t.Errorf("SignIn() error = %v, want ErrCodeVerifierRequired", err)
		}
	})

	t.Run("v005_requires_code_verifier", func(t *testing.T) {
		t.Parallel()
		svc, _, _ := newTestService()
		registerUser(t, svc)

		_, err := svc.SignIn(context.Background(), SignInRequest{
			Email: "user@example.com", Password: "correct-password",
			APIVersion: "005", CodeVerifier: "",
		})
		if !errors.Is(err, ErrCodeVerifierRequired) {
			t.Errorf("SignIn() error = %v, want ErrCodeVerifierRequired", err)
		}
	})

	t.Run("locked_account", func(t *testing.T) {
		t.Parallel()
		svc, users, _ := newTestService()
		registerUser(t, svc)

		// Lock the account
		u := users.byEmail["user@example.com"]
		lockUntil := time.Now().Add(1 * time.Hour)
		u.LockedUntil = &lockUntil
		_ = users.Update(context.Background(), u)

		_, err := svc.SignIn(context.Background(), SignInRequest{
			Email: "user@example.com", Password: "correct-password",
			APIVersion: "004", CodeVerifier: "verifier",
		})
		if !errors.Is(err, ErrAccountLocked) {
			t.Errorf("SignIn() error = %v, want ErrAccountLocked", err)
		}
	})

	t.Run("failed_attempts_increment_counter", func(t *testing.T) {
		t.Parallel()
		svc, users, _ := newTestService()
		registerUser(t, svc)

		_, _ = svc.SignIn(context.Background(), SignInRequest{
			Email: "user@example.com", Password: "wrong",
			APIVersion: "004", CodeVerifier: "verifier",
		})

		u := users.byEmail["user@example.com"]
		if u.NumFailedAttempts == nil || *u.NumFailedAttempts != 1 {
			count := 0
			if u.NumFailedAttempts != nil {
				count = *u.NumFailedAttempts
			}
			t.Errorf("NumFailedAttempts = %d, want 1", count)
		}
	})

	t.Run("success_resets_failed_counter", func(t *testing.T) {
		t.Parallel()
		svc, users, _ := newTestService()
		registerUser(t, svc)

		// Fail once
		_, _ = svc.SignIn(context.Background(), SignInRequest{
			Email: "user@example.com", Password: "wrong",
			APIVersion: "004", CodeVerifier: "verifier",
		})

		// Succeed
		_, _ = svc.SignIn(context.Background(), SignInRequest{
			Email: "user@example.com", Password: "correct-password",
			APIVersion: "004", CodeVerifier: "verifier",
		})

		u := users.byEmail["user@example.com"]
		if u.NumFailedAttempts != nil && *u.NumFailedAttempts != 0 {
			t.Errorf("NumFailedAttempts = %d, want 0 after success", *u.NumFailedAttempts)
		}
	})

	t.Run("creates_new_session", func(t *testing.T) {
		t.Parallel()
		svc, _, sessions := newTestService()
		registerUser(t, svc)

		resp, err := svc.SignIn(context.Background(), SignInRequest{
			Email: "user@example.com", Password: "correct-password",
			APIVersion: "004", CodeVerifier: "verifier",
		})
		if err != nil {
			t.Fatalf("SignIn() error = %v", err)
		}
		if resp.Session.AccessExpiration == 0 {
			t.Error("AccessExpiration is zero")
		}
		if resp.Session.RefreshExpiration == 0 {
			t.Error("RefreshExpiration is zero")
		}
		if len(sessions.sessions) < 2 {
			t.Errorf("expected at least 2 sessions, got %d", len(sessions.sessions))
		}
	})
}

// --- RefreshSession Tests ---

func TestRefreshSession(t *testing.T) {
	t.Parallel()

	t.Run("valid_refresh_token", func(t *testing.T) {
		t.Parallel()
		svc, _, _ := newTestService()
		ctx := context.Background()

		regResp, _ := svc.Register(ctx, RegisterRequest{
			Email: "user@example.com", Password: "pass", APIVersion: "004", Version: "004",
		})

		resp, err := svc.RefreshSession(ctx, RefreshSessionRequest{
			AccessToken:  regResp.Session.AccessToken,
			RefreshToken: regResp.Session.RefreshToken,
		})
		if err != nil {
			t.Fatalf("RefreshSession() error = %v", err)
		}
		if resp.Session.AccessToken == "" {
			t.Error("new access token is empty")
		}
		// New tokens should differ from old
		if resp.Session.AccessToken == regResp.Session.AccessToken {
			t.Error("access token not rotated")
		}
	})

	t.Run("expired_refresh_token", func(t *testing.T) {
		t.Parallel()
		svc, _, sessions := newTestService()
		ctx := context.Background()

		regResp, _ := svc.Register(ctx, RegisterRequest{
			Email: "user@example.com", Password: "pass", APIVersion: "004", Version: "004",
		})

		// Expire the session's refresh token
		for _, s := range sessions.sessions {
			s.RefreshExpiration = time.Now().Add(-1 * time.Hour)
		}

		_, err := svc.RefreshSession(ctx, RefreshSessionRequest{
			AccessToken:  regResp.Session.AccessToken,
			RefreshToken: regResp.Session.RefreshToken,
		})
		if !errors.Is(err, ErrSessionExpired) {
			t.Errorf("RefreshSession() error = %v, want ErrSessionExpired", err)
		}
	})

	t.Run("invalid_token", func(t *testing.T) {
		t.Parallel()
		svc, _, _ := newTestService()

		_, err := svc.RefreshSession(context.Background(), RefreshSessionRequest{
			AccessToken:  "bogus-access",
			RefreshToken: "bogus-refresh",
		})
		if !errors.Is(err, ErrSessionNotFound) {
			t.Errorf("RefreshSession() error = %v, want ErrSessionNotFound", err)
		}
	})

	t.Run("updates_tokens_in_db", func(t *testing.T) {
		t.Parallel()
		svc, _, sessions := newTestService()
		ctx := context.Background()

		regResp, _ := svc.Register(ctx, RegisterRequest{
			Email: "user@example.com", Password: "pass", APIVersion: "004", Version: "004",
		})

		oldHashedAccess := HashToken(regResp.Session.AccessToken)
		resp, _ := svc.RefreshSession(ctx, RefreshSessionRequest{
			AccessToken:  regResp.Session.AccessToken,
			RefreshToken: regResp.Session.RefreshToken,
		})

		// Old token should no longer be findable
		if sessions.byAccessToken[oldHashedAccess] != nil {
			t.Error("old access token still in DB")
		}
		// New token should be findable
		newHashedAccess := HashToken(resp.Session.AccessToken)
		if sessions.byAccessToken[newHashedAccess] == nil {
			t.Error("new access token not in DB")
		}
	})
}

// --- SignOut Tests ---

func TestSignOut(t *testing.T) {
	t.Parallel()

	t.Run("deletes_session", func(t *testing.T) {
		t.Parallel()
		svc, _, sessions := newTestService()
		ctx := context.Background()

		_, _ = svc.Register(ctx, RegisterRequest{
			Email: "user@example.com", Password: "pass", APIVersion: "004", Version: "004",
		})

		// Get the session UUID
		var sessionUUID string
		for id := range sessions.sessions {
			sessionUUID = id
			break
		}

		err := svc.SignOut(ctx, sessionUUID)
		if err != nil {
			t.Fatalf("SignOut() error = %v", err)
		}
		if len(sessions.sessions) != 0 {
			t.Errorf("session count = %d, want 0", len(sessions.sessions))
		}
	})

	t.Run("invalid_session", func(t *testing.T) {
		t.Parallel()
		svc, _, _ := newTestService()
		err := svc.SignOut(context.Background(), "nonexistent-uuid")
		if !errors.Is(err, ErrSessionNotFound) {
			t.Errorf("SignOut() error = %v, want ErrSessionNotFound", err)
		}
	})
}
