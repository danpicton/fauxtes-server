package auth

import (
	"context"
	"net/http"
	"strings"
	"time"
)

type contextKey string

const (
	ContextKeyUserUUID    contextKey = "user_uuid"
	ContextKeySessionUUID contextKey = "session_uuid"
	ContextKeyReadOnly    contextKey = "readonly_access"
)

func UserUUIDFromContext(ctx context.Context) string {
	v, _ := ctx.Value(ContextKeyUserUUID).(string)
	return v
}

func SessionUUIDFromContext(ctx context.Context) string {
	v, _ := ctx.Value(ContextKeySessionUUID).(string)
	return v
}

func ReadOnlyFromContext(ctx context.Context) bool {
	v, _ := ctx.Value(ContextKeyReadOnly).(bool)
	return v
}

// AuthMiddleware validates Bearer tokens and sets user context.
func AuthMiddleware(sessions SessionRepository) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			authHeader := r.Header.Get("Authorization")
			if authHeader == "" {
				http.Error(w, `{"error":{"message":"Invalid login credentials.","tag":"invalid-auth"}}`, http.StatusUnauthorized)
				return
			}

			token := strings.TrimPrefix(authHeader, "Bearer ")
			if token == authHeader {
				http.Error(w, `{"error":{"message":"Invalid login credentials.","tag":"invalid-auth"}}`, http.StatusUnauthorized)
				return
			}

			hashedToken := HashToken(token)
			session, err := sessions.FindByAccessToken(r.Context(), hashedToken)
			if err != nil || session == nil {
				http.Error(w, `{"error":{"message":"Invalid login credentials.","tag":"invalid-auth"}}`, http.StatusUnauthorized)
				return
			}

			if session.AccessExpiration.Before(time.Now()) {
				http.Error(w, `{"error":{"message":"Session expired.","tag":"expired-access-token"}}`, http.StatusUnauthorized)
				return
			}

			ctx := r.Context()
			ctx = context.WithValue(ctx, ContextKeyUserUUID, session.UserUUID)
			ctx = context.WithValue(ctx, ContextKeySessionUUID, session.UUID)
			ctx = context.WithValue(ctx, ContextKeyReadOnly, session.ReadonlyAccess)

			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}
