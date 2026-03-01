package middleware

import (
	"context"
	"net/http"
	"strings"
)

type contextKey string

const principalKey contextKey = "gtd.auth.principal"
const tokenKey contextKey = "gtd.auth.token"

// APIKeyValidator validates an API key token.
type APIKeyValidator interface {
	ValidateAPIKey(ctx context.Context, token string) (bool, error)
}

// Auth enforces API key auth via Bearer token or cookie token.
func Auth(validator APIKeyValidator, cookieName string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if validator == nil {
				http.Error(w, `{"error":"auth is not configured"}`, http.StatusUnauthorized)
				return
			}

			token := extractToken(r, cookieName)
			if token == "" {
				http.Error(w, `{"error":"missing authentication token"}`, http.StatusUnauthorized)
				return
			}
			ok, err := validator.ValidateAPIKey(r.Context(), token)
			if err != nil || !ok {
				http.Error(w, `{"error":"invalid authentication token"}`, http.StatusUnauthorized)
				return
			}

			ctx := context.WithValue(r.Context(), principalKey, "api_key")
			ctx = context.WithValue(ctx, tokenKey, token)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// Principal returns the authenticated principal label.
func Principal(ctx context.Context) string {
	v, _ := ctx.Value(principalKey).(string)
	return v
}

// Token returns the authenticated token captured by auth middleware.
func Token(ctx context.Context) string {
	v, _ := ctx.Value(tokenKey).(string)
	return v
}

func extractToken(r *http.Request, cookieName string) string {
	const bearerPrefix = "Bearer "
	auth := strings.TrimSpace(r.Header.Get("Authorization"))
	if strings.HasPrefix(auth, bearerPrefix) {
		return strings.TrimSpace(strings.TrimPrefix(auth, bearerPrefix))
	}

	if cookieName != "" {
		if c, err := r.Cookie(cookieName); err == nil && strings.TrimSpace(c.Value) != "" {
			return strings.TrimSpace(c.Value)
		}
	}

	if c, err := r.Cookie("access_token"); err == nil && strings.TrimSpace(c.Value) != "" {
		return strings.TrimSpace(c.Value)
	}
	return ""
}

