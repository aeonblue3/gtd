package handlers

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
	apimw "gtd/internal/api/middleware"
)

// AuthHandler exposes authentication route skeletons.
type AuthHandler struct {
	APIKeys         APIKeyManager
	Sessions        SessionManager
	PasswordAuth    PasswordAuthManager
	Events          AuthEventLogger
	TokenCookieName string
	CSRFCookieName  string
	CookieSecure    bool
}

// APIKeyManager defines create/revoke operations for API keys.
type APIKeyManager interface {
	CreateAPIKey(description string) (string, string, error)
	RevokeAPIKey(ctx context.Context, id string) error
	RevokeAPIKeyByToken(ctx context.Context, token string) (string, error)
	RotateAPIKeyByToken(ctx context.Context, token, description string) (string, string, string, error)
}

// SessionManager defines list/revoke operations for auth sessions.
type SessionManager interface {
	ListActiveSessions(ctx context.Context) ([]SessionView, error)
	RevokeSession(ctx context.Context, id string) error
	RevokeAllSessions(ctx context.Context) (int64, error)
	RevokeSessionByAccessToken(ctx context.Context, token string) (string, error)
	RotateSessionByAccessToken(ctx context.Context, token string) (*LoginResult, string, error)
	RotateSessionByRefreshToken(ctx context.Context, refreshToken string) (*LoginResult, string, error)
}

// SessionView is the handler-facing response shape for session objects.
type SessionView struct {
	ID        string `json:"id"`
	UserID    string `json:"user_id"`
	ExpiresAt int64  `json:"expires_at"`
	CreatedAt int64  `json:"created_at"`
	Revoked   bool   `json:"revoked"`
}

// PasswordAuthManager defines password+TOTP setup and login operations.
type PasswordAuthManager interface {
	SetupMFA(ctx context.Context, email, password string) (*MFASetupResult, error)
	VerifyMFA(ctx context.Context, email, code string) error
	Login(ctx context.Context, email, password, totpCode string) (*LoginResult, error)
}

// AuthEventLogger records security-relevant auth events.
type AuthEventLogger interface {
	Record(ctx context.Context, eventType, ipAddress, userAgent string, metadata map[string]any) error
}

// MFASetupResult is returned when TOTP enrollment begins.
type MFASetupResult struct {
	Email      string `json:"email"`
	Secret     string `json:"secret"`
	OTPAuthURL string `json:"otpauth_url"`
}

// LoginResult is returned after successful password+TOTP login.
type LoginResult struct {
	SessionID    string `json:"session_id"`
	UserID       string `json:"user_id"`
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	ExpiresAt    int64  `json:"expires_at"`
}

func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	if h.PasswordAuth == nil {
		writeError(w, http.StatusNotImplemented, "password auth service not configured")
		return
	}
	var in struct {
		Email    string `json:"email"`
		Password string `json:"password"`
		TOTPCode string `json:"totp_code"`
	}
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		writeError(w, http.StatusBadRequest, "invalid json body")
		return
	}

	result, err := h.PasswordAuth.Login(r.Context(), in.Email, in.Password, in.TOTPCode)
	if err != nil {
		h.recordEvent(r, "login_failure", map[string]any{"email": strings.TrimSpace(strings.ToLower(in.Email))})
		writeError(w, http.StatusUnauthorized, err.Error())
		return
	}
	h.recordEvent(r, "login_success", map[string]any{"user_id": result.UserID, "session_id": result.SessionID})

	if h.TokenCookieName != "" {
		setAuthCookie(w, h.TokenCookieName, result.AccessToken, h.CookieSecure)
	}
	setAuthCookie(w, "access_token", result.AccessToken, h.CookieSecure)
	setAuthCookie(w, "refresh_token", result.RefreshToken, h.CookieSecure)
	csrfToken, err := issueCSRFCookie(w, h.CSRFCookieName, h.CookieSecure)
	if err == nil {
		resp := map[string]any{
			"session_id": result.SessionID,
			"user_id":    result.UserID,
			"expires_at": result.ExpiresAt,
			"csrf_token": csrfToken,
		}
		if shouldReturnSessionTokens(r) {
			resp["access_token"] = result.AccessToken
			resp["refresh_token"] = result.RefreshToken
		}
		writeJSON(w, http.StatusOK, resp)
		return
	}
	// If CSRF cookie generation fails, still return successful login response.
	writeJSON(w, http.StatusOK, result)
}
func (h *AuthHandler) Logout(w http.ResponseWriter, r *http.Request) {
	token := strings.TrimSpace(apimw.Token(r.Context()))
	if token == "" {
		writeError(w, http.StatusUnauthorized, "missing authentication token")
		return
	}

	if h.APIKeys != nil {
		revokedID, err := h.APIKeys.RevokeAPIKeyByToken(r.Context(), token)
		if err == nil {
			h.recordEvent(r, "logout_success", map[string]any{"auth_type": "api_key", "revoked_api_key_id": revokedID})
			clearAuthCookies(w, h.TokenCookieName, h.CSRFCookieName, h.CookieSecure)
			writeJSON(w, http.StatusOK, map[string]any{
				"logged_out":         true,
				"revoked_api_key_id": revokedID,
			})
			return
		}
	}
	if h.Sessions != nil {
		revokedID, err := h.Sessions.RevokeSessionByAccessToken(r.Context(), token)
		if err == nil {
			h.recordEvent(r, "logout_success", map[string]any{"auth_type": "session", "revoked_session_id": revokedID})
			clearAuthCookies(w, h.TokenCookieName, h.CSRFCookieName, h.CookieSecure)
			writeJSON(w, http.StatusOK, map[string]any{
				"logged_out":         true,
				"revoked_session_id": revokedID,
			})
			return
		}
	}
	writeError(w, http.StatusUnauthorized, "could not revoke current token")
	h.recordEvent(r, "logout_failure", nil)
}

func (h *AuthHandler) Refresh(w http.ResponseWriter, r *http.Request) {
	var in struct {
		Description  string `json:"description"`
		RefreshToken string `json:"refresh_token"`
	}
	if r.Body != nil {
		_ = json.NewDecoder(r.Body).Decode(&in)
	}
	refreshToken := strings.TrimSpace(in.RefreshToken)
	refreshFromCookie := false
	if refreshToken == "" {
		if c, err := r.Cookie("refresh_token"); err == nil {
			refreshToken = strings.TrimSpace(c.Value)
			refreshFromCookie = refreshToken != ""
		}
	}
	if refreshFromCookie && !hasBearerAuth(r) {
		if err := validateCSRFRequest(r, h.CSRFCookieName); err != nil {
			writeError(w, http.StatusForbidden, err.Error())
			return
		}
	}

	if refreshToken != "" && h.Sessions != nil {
		next, oldID, err := h.Sessions.RotateSessionByRefreshToken(r.Context(), refreshToken)
		if err == nil {
			h.recordEvent(r, "token_refresh", map[string]any{"auth_type": "session", "revoked_session_id": oldID, "session_id": next.SessionID})
			if h.TokenCookieName != "" {
				setAuthCookie(w, h.TokenCookieName, next.AccessToken, h.CookieSecure)
			}
			setAuthCookie(w, "access_token", next.AccessToken, h.CookieSecure)
			setAuthCookie(w, "refresh_token", next.RefreshToken, h.CookieSecure)
			csrfToken, _ := issueCSRFCookie(w, h.CSRFCookieName, h.CookieSecure)
			resp := map[string]any{
				"session_id":         next.SessionID,
				"user_id":            next.UserID,
				"expires_at":         next.ExpiresAt,
				"revoked_session_id": oldID,
				"csrf_token":         csrfToken,
			}
			if shouldReturnSessionTokens(r) {
				resp["access_token"] = next.AccessToken
				resp["refresh_token"] = next.RefreshToken
			}
			writeJSON(w, http.StatusOK, resp)
			return
		}
	}

	if h.APIKeys == nil {
		writeError(w, http.StatusUnauthorized, "refresh token invalid and api key rotation unavailable")
		return
	}
	token := strings.TrimSpace(apimw.Token(r.Context()))
	if token == "" {
		token = parseBearerToken(r.Header.Get("Authorization"))
	}
	if token == "" {
		writeError(w, http.StatusUnauthorized, "missing authentication token or refresh token")
		return
	}

	description := strings.TrimSpace(in.Description)
	if description == "" {
		description = "rotated-api-client"
	}

	if h.APIKeys != nil {
		newID, raw, revokedID, err := h.APIKeys.RotateAPIKeyByToken(r.Context(), token, description)
		if err == nil {
			h.recordEvent(r, "token_refresh", map[string]any{"auth_type": "api_key", "revoked_api_key_id": revokedID, "api_key_id": newID})
			if h.TokenCookieName != "" {
				setAuthCookie(w, h.TokenCookieName, raw, h.CookieSecure)
			}
			setAuthCookie(w, "access_token", raw, h.CookieSecure)
			_, _ = issueCSRFCookie(w, h.CSRFCookieName, h.CookieSecure)

			writeJSON(w, http.StatusOK, map[string]string{
				"id":                 newID,
				"api_key":            raw,
				"description":        description,
				"revoked_api_key_id": revokedID,
			})
			return
		}
	}
	if h.Sessions != nil {
		next, oldID, err := h.Sessions.RotateSessionByAccessToken(r.Context(), token)
		if err == nil {
			h.recordEvent(r, "token_refresh", map[string]any{"auth_type": "session_access", "revoked_session_id": oldID, "session_id": next.SessionID})
			if h.TokenCookieName != "" {
				setAuthCookie(w, h.TokenCookieName, next.AccessToken, h.CookieSecure)
			}
			setAuthCookie(w, "access_token", next.AccessToken, h.CookieSecure)
			setAuthCookie(w, "refresh_token", next.RefreshToken, h.CookieSecure)
			csrfToken, _ := issueCSRFCookie(w, h.CSRFCookieName, h.CookieSecure)
			resp := map[string]any{
				"session_id":         next.SessionID,
				"user_id":            next.UserID,
				"expires_at":         next.ExpiresAt,
				"revoked_session_id": oldID,
				"csrf_token":         csrfToken,
			}
			if shouldReturnSessionTokens(r) {
				resp["access_token"] = next.AccessToken
				resp["refresh_token"] = next.RefreshToken
			}
			writeJSON(w, http.StatusOK, resp)
			return
		}
	}
	writeError(w, http.StatusUnauthorized, "could not rotate current token")
	h.recordEvent(r, "token_refresh_failure", nil)
}
func (h *AuthHandler) SetupMFA(w http.ResponseWriter, r *http.Request) {
	if h.PasswordAuth == nil {
		writeError(w, http.StatusNotImplemented, "password auth service not configured")
		return
	}
	var in struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		writeError(w, http.StatusBadRequest, "invalid json body")
		return
	}

	result, err := h.PasswordAuth.SetupMFA(r.Context(), in.Email, in.Password)
	if err != nil {
		h.recordEvent(r, "mfa_setup_failure", map[string]any{"email": strings.TrimSpace(strings.ToLower(in.Email))})
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	h.recordEvent(r, "mfa_setup_success", map[string]any{"email": result.Email})
	writeJSON(w, http.StatusOK, result)
}

func (h *AuthHandler) VerifyMFA(w http.ResponseWriter, r *http.Request) {
	if h.PasswordAuth == nil {
		writeError(w, http.StatusNotImplemented, "password auth service not configured")
		return
	}
	var in struct {
		Email string `json:"email"`
		Code  string `json:"code"`
	}
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		writeError(w, http.StatusBadRequest, "invalid json body")
		return
	}

	if err := h.PasswordAuth.VerifyMFA(r.Context(), in.Email, in.Code); err != nil {
		h.recordEvent(r, "mfa_verify_failure", map[string]any{"email": strings.TrimSpace(strings.ToLower(in.Email))})
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	h.recordEvent(r, "mfa_verify_success", map[string]any{"email": strings.TrimSpace(strings.ToLower(in.Email))})
	writeJSON(w, http.StatusOK, map[string]bool{"verified": true})
}

// CSRF issues/rotates CSRF token and returns it for SPA bootstrap.
func (h *AuthHandler) CSRF(w http.ResponseWriter, _ *http.Request) {
	csrfToken, err := issueCSRFCookie(w, h.CSRFCookieName, h.CookieSecure)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to issue csrf token")
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"csrf_token": csrfToken})
}
func (h *AuthHandler) ListSessions(w http.ResponseWriter, r *http.Request) {
	if h.Sessions == nil {
		writeError(w, http.StatusNotImplemented, "session service not configured")
		return
	}
	sessions, err := h.Sessions.ListActiveSessions(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, sessions)
}

func (h *AuthHandler) RevokeSession(w http.ResponseWriter, r *http.Request) {
	if h.Sessions == nil {
		writeError(w, http.StatusNotImplemented, "session service not configured")
		return
	}
	id := strings.TrimSpace(chi.URLParam(r, "id"))
	if id == "" {
		writeError(w, http.StatusBadRequest, "missing session id")
		return
	}
	if err := h.Sessions.RevokeSession(r.Context(), id); err != nil {
		writeError(w, http.StatusNotFound, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"revoked": id})
}

func (h *AuthHandler) RevokeAll(w http.ResponseWriter, r *http.Request) {
	if h.Sessions == nil {
		writeError(w, http.StatusNotImplemented, "session service not configured")
		return
	}
	n, err := h.Sessions.RevokeAllSessions(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]int64{"revoked_count": n})
}

func (h *AuthHandler) CreateAPIKey(w http.ResponseWriter, r *http.Request) {
	if h.APIKeys == nil {
		writeError(w, http.StatusNotImplemented, "api key service not configured")
		return
	}

	var in struct {
		Description string `json:"description"`
	}
	if r.Body != nil {
		_ = json.NewDecoder(r.Body).Decode(&in)
	}
	description := strings.TrimSpace(in.Description)
	if description == "" {
		description = "api-client"
	}

	id, raw, err := h.APIKeys.CreateAPIKey(description)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, map[string]string{
		"id":          id,
		"api_key":     raw,
		"description": description,
	})
}

func (h *AuthHandler) RevokeAPIKey(w http.ResponseWriter, r *http.Request) {
	if h.APIKeys == nil {
		writeError(w, http.StatusNotImplemented, "api key service not configured")
		return
	}
	id := strings.TrimSpace(chi.URLParam(r, "id"))
	if id == "" {
		writeError(w, http.StatusBadRequest, "missing api key id")
		return
	}
	if err := h.APIKeys.RevokeAPIKey(r.Context(), id); err != nil {
		writeError(w, http.StatusNotFound, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"revoked": id})
}

func clearAuthCookies(w http.ResponseWriter, tokenName, csrfName string, secure bool) {
	if tokenName != "" {
		http.SetCookie(w, &http.Cookie{
			Name:     tokenName,
			Value:    "",
			Path:     "/",
			HttpOnly: true,
			MaxAge:   -1,
			Secure:   secure,
			SameSite: http.SameSiteStrictMode,
		})
	}
	http.SetCookie(w, &http.Cookie{
		Name:     "access_token",
		Value:    "",
		Path:     "/",
		HttpOnly: true,
		MaxAge:   -1,
		Secure:   secure,
		SameSite: http.SameSiteStrictMode,
	})
	http.SetCookie(w, &http.Cookie{
		Name:     "refresh_token",
		Value:    "",
		Path:     "/",
		HttpOnly: true,
		MaxAge:   -1,
		Secure:   secure,
		SameSite: http.SameSiteStrictMode,
	})
	if strings.TrimSpace(csrfName) == "" {
		csrfName = "gtd_csrf"
	}
	http.SetCookie(w, &http.Cookie{
		Name:     csrfName,
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		Secure:   secure,
		SameSite: http.SameSiteStrictMode,
	})
}

func setAuthCookie(w http.ResponseWriter, name, value string, secure bool) {
	http.SetCookie(w, &http.Cookie{
		Name:     name,
		Value:    value,
		Path:     "/",
		HttpOnly: true,
		Secure:   secure,
		SameSite: http.SameSiteStrictMode,
	})
}

func issueCSRFCookie(w http.ResponseWriter, name string, secure bool) (string, error) {
	if strings.TrimSpace(name) == "" {
		name = "gtd_csrf"
	}
	token, err := randomToken()
	if err != nil {
		return "", err
	}
	http.SetCookie(w, &http.Cookie{
		Name:     name,
		Value:    token,
		Path:     "/",
		HttpOnly: false,
		Secure:   secure,
		SameSite: http.SameSiteStrictMode,
	})
	return token, nil
}

func parseBearerToken(value string) string {
	const prefix = "Bearer "
	value = strings.TrimSpace(value)
	if !strings.HasPrefix(value, prefix) {
		return ""
	}
	return strings.TrimSpace(strings.TrimPrefix(value, prefix))
}

func hasBearerAuth(r *http.Request) bool {
	return parseBearerToken(r.Header.Get("Authorization")) != ""
}

func shouldReturnSessionTokens(r *http.Request) bool {
	if hasBearerAuth(r) {
		return true
	}
	return strings.EqualFold(strings.TrimSpace(r.Header.Get("X-Return-Tokens")), "true")
}

func validateCSRFRequest(r *http.Request, csrfCookie string) error {
	if strings.TrimSpace(csrfCookie) == "" {
		csrfCookie = "gtd_csrf"
	}
	c, err := r.Cookie(csrfCookie)
	if err != nil || strings.TrimSpace(c.Value) == "" {
		return fmt.Errorf("missing csrf cookie")
	}
	header := strings.TrimSpace(r.Header.Get("X-CSRF-Token"))
	if header == "" || header != c.Value {
		return fmt.Errorf("invalid csrf token")
	}
	return nil
}

func randomToken() (string, error) {
	buf := make([]byte, 32)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(buf), nil
}

func (h *AuthHandler) recordEvent(r *http.Request, eventType string, metadata map[string]any) {
	if h.Events == nil {
		return
	}
	_ = h.Events.Record(r.Context(), eventType, clientIP(r), r.UserAgent(), metadata)
}

func clientIP(r *http.Request) string {
	host, _, err := net.SplitHostPort(strings.TrimSpace(r.RemoteAddr))
	if err == nil && host != "" {
		return host
	}
	return strings.TrimSpace(r.RemoteAddr)
}
