package auth

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/danpicton/fauxtes-server/internal/domain"
)

var (
	ErrUserNotFound         = errors.New("user not found")
	ErrDuplicateEmail       = errors.New("email already registered")
	ErrInvalidEmail         = errors.New("invalid email")
	ErrInvalidAPIVersion    = errors.New("invalid API version")
	ErrLegacyAPIVersion     = errors.New("legacy API version not supported for registration")
	ErrRegistrationDisabled = errors.New("registration is disabled")
	ErrInvalidCredentials   = errors.New("invalid email or password")
	ErrAccountLocked        = errors.New("account is locked")
	ErrCodeVerifierRequired = errors.New("please update your client application")
	ErrInvalidCodeVerifier  = errors.New("invalid email or password")
	ErrSessionNotFound      = errors.New("session not found")
	ErrSessionExpired       = errors.New("session expired")
)

type ServiceConfig struct {
	DisableRegistration  bool
	AccessTokenLifetime  time.Duration
	RefreshTokenLifetime time.Duration
}

func DefaultServiceConfig() ServiceConfig {
	return ServiceConfig{
		AccessTokenLifetime:  24 * time.Hour,
		RefreshTokenLifetime: 30 * 24 * time.Hour,
	}
}

type Service struct {
	users    UserRepository
	sessions SessionRepository
	config   ServiceConfig
}

func NewService(users UserRepository, sessions SessionRepository, config ServiceConfig) *Service {
	return &Service{
		users:    users,
		sessions: sessions,
		config:   config,
	}
}

type RegisterRequest struct {
	Email      string `json:"email"`
	Password   string `json:"password"`
	APIVersion string `json:"api"`
	PwNonce    string `json:"pw_nonce"`
	PwCost     int    `json:"pw_cost,omitempty"`
	PwSalt     string `json:"pw_salt,omitempty"`
	PwAlg      string `json:"pw_alg,omitempty"`
	PwFunc     string `json:"pw_func,omitempty"`
	PwKeySize  int    `json:"pw_key_size,omitempty"`
	Version    string `json:"version"`
	Origination string `json:"origination,omitempty"`
	Created    string `json:"created,omitempty"`

	// Client metadata
	UserAgent   string `json:"-"`
	Application string `json:"application,omitempty"`
	SNJS        string `json:"snjs,omitempty"`

	// PKCE
	CodeChallenge string `json:"code_challenge,omitempty"`
}

type AuthResponse struct {
	User      AuthResponseUser      `json:"user"`
	Token     string                `json:"token"`
	Session   AuthResponseSession   `json:"session,omitempty"`
	KeyParams domain.KeyParams      `json:"key_params"`
}

type AuthResponseUser struct {
	UUID  string `json:"uuid"`
	Email string `json:"email"`
}

type AuthResponseSession struct {
	AccessToken        string `json:"access_token"`
	RefreshToken       string `json:"refresh_token"`
	AccessExpiration   int64  `json:"access_expiration"`
	RefreshExpiration  int64  `json:"refresh_expiration"`
	ReadonlyAccess     bool   `json:"readonly_access"`
}

func (s *Service) Register(ctx context.Context, req RegisterRequest) (*AuthResponse, error) {
	if s.config.DisableRegistration {
		return nil, ErrRegistrationDisabled
	}

	if req.Email == "" {
		return nil, ErrInvalidEmail
	}

	if !isValidAPIVersion(req.APIVersion) {
		return nil, ErrInvalidAPIVersion
	}

	if isLegacyAPIVersion(req.APIVersion) {
		return nil, ErrLegacyAPIVersion
	}

	existing, err := s.users.FindByEmail(ctx, req.Email)
	if err != nil {
		return nil, fmt.Errorf("checking existing user: %w", err)
	}
	if existing != nil {
		return nil, ErrDuplicateEmail
	}

	hash, err := HashPassword(req.Password)
	if err != nil {
		return nil, fmt.Errorf("hashing password: %w", err)
	}

	serverKey, err := GenerateEncryptedServerKey()
	if err != nil {
		return nil, fmt.Errorf("generating server key: %w", err)
	}

	user := domain.NewUser(req.Email)
	user.EncryptedPassword = hash
	user.EncryptedServerKey = &serverKey
	user.ServerEncryptionVersion = 1

	version := req.Version
	if version == "" {
		version = req.APIVersion
	}
	user.Version = &version

	if req.PwNonce != "" {
		user.PwNonce = &req.PwNonce
	}
	if req.PwCost != 0 {
		user.PwCost = &req.PwCost
	}
	if req.PwSalt != "" {
		user.PwSalt = &req.PwSalt
	}
	if req.PwAlg != "" {
		user.PwAlg = &req.PwAlg
	}
	if req.PwFunc != "" {
		user.PwFunc = &req.PwFunc
	}
	if req.PwKeySize != 0 {
		user.PwKeySize = &req.PwKeySize
	}
	if req.Origination != "" {
		user.KpOrigination = &req.Origination
	}
	if req.Created != "" {
		user.KpCreated = &req.Created
	}

	if err := s.users.Create(ctx, &user); err != nil {
		return nil, fmt.Errorf("creating user: %w", err)
	}

	return s.createSessionResponse(ctx, &user, req.UserAgent, req.Application, req.SNJS, req.APIVersion, false)
}

type GetParamsRequest struct {
	Email         string `json:"email"`
	Authenticated bool
}

func (s *Service) GetParams(ctx context.Context, req GetParamsRequest) (*domain.KeyParams, error) {
	if req.Email == "" {
		return nil, ErrInvalidEmail
	}

	user, err := s.users.FindByEmail(ctx, req.Email)
	if err != nil {
		return nil, fmt.Errorf("finding user: %w", err)
	}

	if user == nil {
		kp := domain.PseudoKeyParams(req.Email)
		return &kp, nil
	}

	kp := domain.KeyParamsFromUser(*user, req.Authenticated)
	return &kp, nil
}

type SignInRequest struct {
	Email        string `json:"email"`
	Password     string `json:"password"`
	APIVersion   string `json:"api"`
	CodeVerifier string `json:"code_verifier,omitempty"`

	// Client metadata
	UserAgent   string `json:"-"`
	Application string `json:"application,omitempty"`
	SNJS        string `json:"snjs,omitempty"`
}

func (s *Service) SignIn(ctx context.Context, req SignInRequest) (*AuthResponse, error) {
	if req.Email == "" {
		return nil, ErrInvalidEmail
	}

	if !isValidAPIVersion(req.APIVersion) {
		return nil, ErrInvalidAPIVersion
	}

	// V004+ requires PKCE code verifier
	if requiresCodeVerifier(req.APIVersion) && req.CodeVerifier == "" {
		return nil, ErrCodeVerifierRequired
	}

	user, err := s.users.FindByEmail(ctx, req.Email)
	if err != nil {
		return nil, fmt.Errorf("finding user: %w", err)
	}
	if user == nil {
		return nil, ErrInvalidCredentials
	}

	// Check account lock
	if user.LockedUntil != nil && user.LockedUntil.After(time.Now()) {
		return nil, ErrAccountLocked
	}

	// Verify password
	if !CheckPassword(user.EncryptedPassword, req.Password) {
		s.incrementFailedAttempts(ctx, user)
		return nil, ErrInvalidCredentials
	}

	// Verify PKCE code verifier if required
	// The SN server stores the code_challenge at registration or key params time
	// For simplicity we skip PKCE validation for now if no challenge stored
	// TODO: implement full PKCE flow

	// Reset failed attempts on success
	if user.NumFailedAttempts != nil && *user.NumFailedAttempts > 0 {
		zero := 0
		user.NumFailedAttempts = &zero
		user.LockedUntil = nil
		user.UpdatedAt = time.Now().UTC()
		_ = s.users.Update(ctx, user)
	}

	return s.createSessionResponse(ctx, user, req.UserAgent, req.Application, req.SNJS, req.APIVersion, false)
}

type RefreshSessionRequest struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
}

func (s *Service) RefreshSession(ctx context.Context, req RefreshSessionRequest) (*AuthResponse, error) {
	hashedRefresh := HashToken(req.RefreshToken)
	session, err := s.sessions.FindByRefreshToken(ctx, hashedRefresh)
	if err != nil {
		return nil, fmt.Errorf("finding session: %w", err)
	}
	if session == nil {
		return nil, ErrSessionNotFound
	}

	if session.RefreshExpiration.Before(time.Now()) {
		return nil, ErrSessionExpired
	}

	user, err := s.users.FindByUUID(ctx, session.UserUUID)
	if err != nil {
		return nil, fmt.Errorf("finding user: %w", err)
	}
	if user == nil {
		return nil, ErrUserNotFound
	}

	// Generate new tokens
	accessToken, refreshToken, err := GenerateSessionTokens()
	if err != nil {
		return nil, fmt.Errorf("generating tokens: %w", err)
	}

	now := time.Now().UTC()
	session.HashedAccessToken = HashToken(accessToken)
	session.HashedRefreshToken = HashToken(refreshToken)
	session.AccessExpiration = now.Add(s.config.AccessTokenLifetime)
	session.RefreshExpiration = now.Add(s.config.RefreshTokenLifetime)
	session.UpdatedAt = now

	if err := s.sessions.Update(ctx, session); err != nil {
		return nil, fmt.Errorf("updating session: %w", err)
	}

	kp := domain.KeyParamsFromUser(*user, true)

	return &AuthResponse{
		User: AuthResponseUser{UUID: user.UUID, Email: user.Email},
		Session: AuthResponseSession{
			AccessToken:       accessToken,
			RefreshToken:      refreshToken,
			AccessExpiration:  session.AccessExpiration.UnixMilli(),
			RefreshExpiration: session.RefreshExpiration.UnixMilli(),
			ReadonlyAccess:    session.ReadonlyAccess,
		},
		KeyParams: kp,
	}, nil
}

func (s *Service) SignOut(ctx context.Context, sessionUUID string) error {
	session, err := s.sessions.FindByUUID(ctx, sessionUUID)
	if err != nil {
		return fmt.Errorf("finding session: %w", err)
	}
	if session == nil {
		return ErrSessionNotFound
	}
	return s.sessions.Delete(ctx, sessionUUID)
}

func (s *Service) createSessionResponse(ctx context.Context, user *domain.User, userAgent, application, snjs, apiVersion string, readonly bool) (*AuthResponse, error) {
	accessToken, refreshToken, err := GenerateSessionTokens()
	if err != nil {
		return nil, fmt.Errorf("generating tokens: %w", err)
	}

	now := time.Now().UTC()
	session := domain.NewSession(user.UUID)
	session.HashedAccessToken = HashToken(accessToken)
	session.HashedRefreshToken = HashToken(refreshToken)
	session.AccessExpiration = now.Add(s.config.AccessTokenLifetime)
	session.RefreshExpiration = now.Add(s.config.RefreshTokenLifetime)
	session.ReadonlyAccess = readonly

	if userAgent != "" {
		session.UserAgent = &userAgent
	}
	if application != "" {
		session.Application = &application
	}
	if snjs != "" {
		session.SNJS = &snjs
	}
	if apiVersion != "" {
		session.APIVersion = &apiVersion
	}

	if err := s.sessions.Create(ctx, &session); err != nil {
		return nil, fmt.Errorf("creating session: %w", err)
	}

	kp := domain.KeyParamsFromUser(*user, true)

	return &AuthResponse{
		User: AuthResponseUser{UUID: user.UUID, Email: user.Email},
		Token: accessToken,
		Session: AuthResponseSession{
			AccessToken:       accessToken,
			RefreshToken:      refreshToken,
			AccessExpiration:  session.AccessExpiration.UnixMilli(),
			RefreshExpiration: session.RefreshExpiration.UnixMilli(),
			ReadonlyAccess:    readonly,
		},
		KeyParams: kp,
	}, nil
}

func (s *Service) incrementFailedAttempts(ctx context.Context, user *domain.User) {
	count := 0
	if user.NumFailedAttempts != nil {
		count = *user.NumFailedAttempts
	}
	count++
	user.NumFailedAttempts = &count

	// Lock after 6 failed attempts
	if count >= 6 {
		lockUntil := time.Now().UTC().Add(30 * time.Minute)
		user.LockedUntil = &lockUntil
	}

	user.UpdatedAt = time.Now().UTC()
	_ = s.users.Update(ctx, user)
}

func isValidAPIVersion(v string) bool {
	switch v {
	case "001", "002", "003", "004", "005", "20190520", "20200115":
		return true
	}
	return false
}

func isLegacyAPIVersion(v string) bool {
	switch v {
	case "001", "002", "003", "20190520":
		return true
	}
	return false
}

func requiresCodeVerifier(v string) bool {
	switch v {
	case "004", "005":
		return true
	}
	return false
}
