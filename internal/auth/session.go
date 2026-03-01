package auth

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/google/uuid"
)

// Session is an API-safe view of an auth session.
type Session struct {
	ID        string `json:"id"`
	UserID    string `json:"user_id"`
	ExpiresAt int64  `json:"expires_at"`
	CreatedAt int64  `json:"created_at"`
	Revoked   bool   `json:"revoked"`
}

// SessionTokens contains generated session token material.
type SessionTokens struct {
	ID           string
	UserID       string
	AccessToken  string
	RefreshToken string
	ExpiresAt    int64
}

// SessionService manages auth session lifecycle operations.
type SessionService struct {
	db *sql.DB
}

// NewSessionService creates a DB-backed session service.
func NewSessionService(db *sql.DB) *SessionService {
	return &SessionService{db: db}
}

// CreateSession inserts a new auth session row with random tokens.
func (s *SessionService) CreateSession(ctx context.Context, userID string, accessTTL, refreshTTL time.Duration) (*SessionTokens, error) {
	accessToken, err := randomToken()
	if err != nil {
		return nil, err
	}
	refreshToken, err := randomToken()
	if err != nil {
		return nil, err
	}
	id := uuid.NewString()
	expiresAt := time.Now().Add(accessTTL).Unix()
	refreshExpiresAt := time.Now().Add(refreshTTL).Unix()
	createdAt := time.Now().Unix()
	_, err = s.db.ExecContext(ctx, `
INSERT INTO auth_sessions (id, user_id, access_token, refresh_token, expires_at, refresh_expires_at, created_at, revoked)
VALUES (?, ?, ?, ?, ?, ?, ?, 0)`,
		id, userID, accessToken, refreshToken, expiresAt, refreshExpiresAt, createdAt)
	if err != nil {
		return nil, fmt.Errorf("create session: %w", err)
	}
	return &SessionTokens{
		ID:           id,
		UserID:       userID,
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		ExpiresAt:    expiresAt,
	}, nil
}

// ValidateAccessToken checks if an access token belongs to an active non-expired session.
func (s *SessionService) ValidateAccessToken(ctx context.Context, token string) (bool, error) {
	_, _ = s.PruneExpiredSessions(ctx)
	row := s.db.QueryRowContext(ctx, `
SELECT COUNT(1)
FROM auth_sessions
WHERE access_token = ? AND revoked = 0 AND expires_at > ?`, token, time.Now().Unix())
	var n int
	if err := row.Scan(&n); err != nil {
		return false, err
	}
	return n > 0, nil
}

// RevokeSessionByAccessToken revokes the session identified by an access token.
func (s *SessionService) RevokeSessionByAccessToken(ctx context.Context, token string) (string, error) {
	_, _ = s.PruneExpiredSessions(ctx)
	row := s.db.QueryRowContext(ctx, `SELECT id FROM auth_sessions WHERE access_token = ? AND revoked = 0 LIMIT 1`, token)
	var id string
	if err := row.Scan(&id); err != nil {
		if err == sql.ErrNoRows {
			return "", fmt.Errorf("session not found for token")
		}
		return "", err
	}
	if err := s.RevokeSession(ctx, id); err != nil {
		return "", err
	}
	return id, nil
}

// RotateSessionByAccessToken rotates the current session token pair.
func (s *SessionService) RotateSessionByAccessToken(ctx context.Context, token string, accessTTL, refreshTTL time.Duration) (*SessionTokens, string, error) {
	_, _ = s.PruneExpiredSessions(ctx)
	row := s.db.QueryRowContext(ctx, `SELECT id, user_id FROM auth_sessions WHERE access_token = ? AND revoked = 0 LIMIT 1`, token)
	var oldID, userID string
	if err := row.Scan(&oldID, &userID); err != nil {
		if err == sql.ErrNoRows {
			return nil, "", fmt.Errorf("session not found for token")
		}
		return nil, "", err
	}
	next, err := s.CreateSession(ctx, userID, accessTTL, refreshTTL)
	if err != nil {
		return nil, "", err
	}
	if err := s.RevokeSession(ctx, oldID); err != nil {
		return nil, "", err
	}
	return next, oldID, nil
}

// RotateSessionByRefreshToken rotates session tokens using a refresh token.
func (s *SessionService) RotateSessionByRefreshToken(ctx context.Context, refreshToken string, accessTTL, refreshTTL time.Duration) (*SessionTokens, string, error) {
	_, _ = s.PruneExpiredSessions(ctx)
	row := s.db.QueryRowContext(ctx, `
SELECT id, user_id
FROM auth_sessions
WHERE refresh_token = ? AND revoked = 0 AND refresh_expires_at > ?
LIMIT 1`, refreshToken, time.Now().Unix())
	var oldID, userID string
	if err := row.Scan(&oldID, &userID); err != nil {
		if err == sql.ErrNoRows {
			return nil, "", fmt.Errorf("session not found for refresh token")
		}
		return nil, "", err
	}
	next, err := s.CreateSession(ctx, userID, accessTTL, refreshTTL)
	if err != nil {
		return nil, "", err
	}
	if err := s.RevokeSession(ctx, oldID); err != nil {
		return nil, "", err
	}
	return next, oldID, nil
}

// ListActiveSessions returns all non-revoked sessions.
func (s *SessionService) ListActiveSessions(ctx context.Context) ([]Session, error) {
	_, _ = s.PruneExpiredSessions(ctx)
	rows, err := s.db.QueryContext(ctx, `
SELECT id, user_id, expires_at, created_at, revoked
FROM auth_sessions
WHERE revoked = 0 AND expires_at > ?
ORDER BY created_at DESC`, time.Now().Unix())
	if err != nil {
		return nil, fmt.Errorf("list sessions: %w", err)
	}
	defer rows.Close()

	var sessions []Session
	for rows.Next() {
		var item Session
		var revokedInt int
		if err := rows.Scan(&item.ID, &item.UserID, &item.ExpiresAt, &item.CreatedAt, &revokedInt); err != nil {
			return nil, fmt.Errorf("scan session: %w", err)
		}
		item.Revoked = revokedInt != 0
		sessions = append(sessions, item)
	}
	return sessions, rows.Err()
}

// RevokeSession revokes one session by ID.
func (s *SessionService) RevokeSession(ctx context.Context, id string) error {
	res, err := s.db.ExecContext(ctx, `UPDATE auth_sessions SET revoked = 1 WHERE id = ?`, id)
	if err != nil {
		return fmt.Errorf("revoke session: %w", err)
	}
	n, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("revoke session rows: %w", err)
	}
	if n == 0 {
		return fmt.Errorf("session not found: %s", id)
	}
	return nil
}

// RevokeAllSessions revokes every active session.
func (s *SessionService) RevokeAllSessions(ctx context.Context) (int64, error) {
	_, _ = s.PruneExpiredSessions(ctx)
	res, err := s.db.ExecContext(ctx, `UPDATE auth_sessions SET revoked = 1 WHERE revoked = 0`)
	if err != nil {
		return 0, fmt.Errorf("revoke all sessions: %w", err)
	}
	n, err := res.RowsAffected()
	if err != nil {
		return 0, fmt.Errorf("revoke all sessions rows: %w", err)
	}
	return n, nil
}

// PruneExpiredSessions deletes sessions whose access lifetime has expired.
func (s *SessionService) PruneExpiredSessions(ctx context.Context) (int64, error) {
	res, err := s.db.ExecContext(ctx, `DELETE FROM auth_sessions WHERE expires_at <= ? AND (refresh_expires_at IS NULL OR refresh_expires_at <= ?)`, time.Now().Unix(), time.Now().Unix())
	if err != nil {
		return 0, fmt.Errorf("prune expired sessions: %w", err)
	}
	n, err := res.RowsAffected()
	if err != nil {
		return 0, fmt.Errorf("prune expired sessions rows: %w", err)
	}
	return n, nil
}
