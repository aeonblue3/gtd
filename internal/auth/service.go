package auth

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/base64"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/pquerna/otp"
	"github.com/pquerna/otp/totp"
	"golang.org/x/crypto/bcrypt"
)

// LoginResult is returned after successful password+TOTP login.
type LoginResult struct {
	SessionID    string `json:"session_id"`
	UserID       string `json:"user_id"`
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	ExpiresAt    int64  `json:"expires_at"`
}

// MFASetupResult contains generated TOTP enrollment data.
type MFASetupResult struct {
	Email     string `json:"email"`
	Secret    string `json:"secret"`
	OTPAuthURL string `json:"otpauth_url"`
}

// Service combines API key, session, and password+TOTP auth operations.
type Service struct {
	db         *sql.DB
	apiKeys    *APIKeyService
	sessions   *SessionService
	events     *EventService
	totpIssuer string
}

// NewService builds the unified auth service.
func NewService(db *sql.DB, totpIssuer string) *Service {
	if strings.TrimSpace(totpIssuer) == "" {
		totpIssuer = "GTD"
	}
	return &Service{
		db:         db,
		apiKeys:    NewAPIKeyService(db),
		sessions:   NewSessionService(db),
		events:     NewEventService(db),
		totpIssuer: totpIssuer,
	}
}

// ValidateAPIKey validates either access session tokens or API keys.
func (s *Service) ValidateAPIKey(ctx context.Context, token string) (bool, error) {
	ok, err := s.sessions.ValidateAccessToken(ctx, token)
	if err != nil {
		return false, err
	}
	if ok {
		return true, nil
	}
	return s.apiKeys.ValidateAPIKey(ctx, token)
}

func (s *Service) CreateAPIKey(description string) (string, string, error) {
	return s.apiKeys.CreateAPIKey(description)
}

func (s *Service) RevokeAPIKey(ctx context.Context, id string) error {
	return s.apiKeys.RevokeAPIKey(ctx, id)
}

func (s *Service) RevokeAPIKeyByToken(ctx context.Context, token string) (string, error) {
	return s.apiKeys.RevokeAPIKeyByToken(ctx, token)
}

func (s *Service) RotateAPIKeyByToken(ctx context.Context, token, description string) (string, string, string, error) {
	return s.apiKeys.RotateAPIKeyByToken(ctx, token, description)
}

func (s *Service) ActiveKeyCount(ctx context.Context) (int, error) {
	return s.apiKeys.ActiveKeyCount(ctx)
}

func (s *Service) Record(ctx context.Context, eventType, ipAddress, userAgent string, metadata map[string]any) error {
	return s.events.Record(ctx, eventType, ipAddress, userAgent, metadata)
}

func (s *Service) ListActiveSessions(ctx context.Context) ([]Session, error) {
	return s.sessions.ListActiveSessions(ctx)
}

func (s *Service) RevokeSession(ctx context.Context, id string) error {
	return s.sessions.RevokeSession(ctx, id)
}

func (s *Service) RevokeAllSessions(ctx context.Context) (int64, error) {
	return s.sessions.RevokeAllSessions(ctx)
}

func (s *Service) RevokeSessionByAccessToken(ctx context.Context, token string) (string, error) {
	return s.sessions.RevokeSessionByAccessToken(ctx, token)
}

func (s *Service) RotateSessionByAccessToken(ctx context.Context, token string) (*SessionTokens, string, error) {
	return s.sessions.RotateSessionByAccessToken(ctx, token, 15*time.Minute, 30*24*time.Hour)
}

func (s *Service) RotateSessionByRefreshToken(ctx context.Context, refreshToken string) (*SessionTokens, string, error) {
	return s.sessions.RotateSessionByRefreshToken(ctx, refreshToken, 15*time.Minute, 30*24*time.Hour)
}

// SetupMFA creates or resets a local user with a new TOTP secret.
func (s *Service) SetupMFA(ctx context.Context, email, password string) (*MFASetupResult, error) {
	email = strings.TrimSpace(strings.ToLower(email))
	password = strings.TrimSpace(password)
	if email == "" || password == "" {
		return nil, fmt.Errorf("email and password are required")
	}

	key, err := totp.Generate(totp.GenerateOpts{
		Issuer:      s.totpIssuer,
		AccountName: email,
	})
	if err != nil {
		return nil, fmt.Errorf("generate totp secret: %w", err)
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return nil, fmt.Errorf("hash password: %w", err)
	}

	row := s.db.QueryRowContext(ctx, `SELECT id FROM users WHERE email = ? LIMIT 1`, email)
	var existingID string
	switch err := row.Scan(&existingID); err {
	case nil:
		_, err = s.db.ExecContext(ctx, `
UPDATE users
SET password_hash = ?, totp_secret = ?, totp_enabled = 0
WHERE id = ?`, string(hash), key.Secret(), existingID)
		if err != nil {
			return nil, fmt.Errorf("update user setup: %w", err)
		}
	case sql.ErrNoRows:
		_, err = s.db.ExecContext(ctx, `
INSERT INTO users (id, email, password_hash, totp_secret, totp_enabled, created_at)
VALUES (?, ?, ?, ?, 0, ?)`, uuid.New().String(), email, string(hash), key.Secret(), time.Now().Unix())
		if err != nil {
			return nil, fmt.Errorf("create setup user: %w", err)
		}
	default:
		return nil, fmt.Errorf("query user: %w", err)
	}

	return &MFASetupResult{
		Email:      email,
		Secret:     key.Secret(),
		OTPAuthURL: key.URL(),
	}, nil
}

// VerifyMFA verifies code and enables TOTP for the user.
func (s *Service) VerifyMFA(ctx context.Context, email, code string) error {
	email = strings.TrimSpace(strings.ToLower(email))
	code = strings.TrimSpace(code)
	if email == "" || code == "" {
		return fmt.Errorf("email and code are required")
	}
	var (
		userID string
		secret sql.NullString
	)
	row := s.db.QueryRowContext(ctx, `SELECT id, totp_secret FROM users WHERE email = ? LIMIT 1`, email)
	if err := row.Scan(&userID, &secret); err != nil {
		if err == sql.ErrNoRows {
			return fmt.Errorf("user not found")
		}
		return fmt.Errorf("query user: %w", err)
	}
	if !secret.Valid || strings.TrimSpace(secret.String) == "" {
		return fmt.Errorf("mfa is not set up for this user")
	}
	if !validateTOTPCode(secret.String, code) {
		return fmt.Errorf("invalid mfa code (check device/system clock and try again)")
	}
	_, err := s.db.ExecContext(ctx, `UPDATE users SET totp_enabled = 1 WHERE id = ?`, userID)
	if err != nil {
		return fmt.Errorf("enable mfa: %w", err)
	}
	return nil
}

// Login validates password+TOTP and creates an auth session.
func (s *Service) Login(ctx context.Context, email, password, totpCode string) (*LoginResult, error) {
	email = strings.TrimSpace(strings.ToLower(email))
	password = strings.TrimSpace(password)
	totpCode = strings.TrimSpace(totpCode)
	if email == "" || password == "" || totpCode == "" {
		return nil, fmt.Errorf("email, password, and totp_code are required")
	}

	var (
		userID       string
		passwordHash string
		secret       sql.NullString
		enabled      int
	)
	row := s.db.QueryRowContext(ctx, `
SELECT id, password_hash, totp_secret, totp_enabled
FROM users
WHERE email = ?
LIMIT 1`, email)
	if err := row.Scan(&userID, &passwordHash, &secret, &enabled); err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("invalid credentials")
		}
		return nil, fmt.Errorf("query user: %w", err)
	}
	if bcrypt.CompareHashAndPassword([]byte(passwordHash), []byte(password)) != nil {
		return nil, fmt.Errorf("invalid credentials")
	}
	if enabled == 0 || !secret.Valid || strings.TrimSpace(secret.String) == "" {
		return nil, fmt.Errorf("mfa is not enabled")
	}
	if !validateTOTPCode(secret.String, totpCode) {
		return nil, fmt.Errorf("invalid mfa code")
	}

	session, err := s.sessions.CreateSession(ctx, userID, 15*time.Minute, 30*24*time.Hour)
	if err != nil {
		return nil, err
	}
	return &LoginResult{
		SessionID:    session.ID,
		UserID:       userID,
		AccessToken:  session.AccessToken,
		RefreshToken: session.RefreshToken,
		ExpiresAt:    session.ExpiresAt,
	}, nil
}

func randomToken() (string, error) {
	buf := make([]byte, 32)
	if _, err := rand.Read(buf); err != nil {
		return "", fmt.Errorf("generate token: %w", err)
	}
	return base64.RawURLEncoding.EncodeToString(buf), nil
}

func validateTOTPCode(secret, code string) bool {
	// Allow +/- 1 time window to handle minor clock skew.
	ok, err := totp.ValidateCustom(
		code,
		secret,
		time.Now(),
		totp.ValidateOpts{
			Period:    30,
			Skew:      1,
			Digits:    otp.DigitsSix,
			Algorithm: otp.AlgorithmSHA1,
		},
	)
	return err == nil && ok
}

