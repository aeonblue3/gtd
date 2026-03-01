package auth

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/base64"
	"fmt"
	"time"

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
)

// APIKeyService manages API key creation and validation.
type APIKeyService struct {
	db *sql.DB
}

// NewAPIKeyService creates an API key service with database backing.
func NewAPIKeyService(db *sql.DB) *APIKeyService {
	return &APIKeyService{db: db}
}

// CreateAPIKey generates and stores a hashed API key, returning the plaintext once.
func (s *APIKeyService) CreateAPIKey(description string) (string, string, error) {
	raw, err := generateAPIKey()
	if err != nil {
		return "", "", err
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(raw), bcrypt.DefaultCost)
	if err != nil {
		return "", "", fmt.Errorf("hash api key: %w", err)
	}

	id := uuid.New().String()
	_, err = s.db.Exec(
		`INSERT INTO api_keys (id, key_hash, description, created_at, revoked) VALUES (?, ?, ?, ?, 0)`,
		id, string(hash), description, time.Now().Unix(),
	)
	if err != nil {
		return "", "", fmt.Errorf("store api key: %w", err)
	}
	return id, raw, nil
}

// ValidateAPIKey validates bearer/cookie API key tokens.
func (s *APIKeyService) ValidateAPIKey(ctx context.Context, token string) (bool, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT id, key_hash FROM api_keys WHERE revoked = 0`)
	if err != nil {
		return false, err
	}
	defer rows.Close()

	for rows.Next() {
		var id, hash string
		if err := rows.Scan(&id, &hash); err != nil {
			return false, err
		}
		if err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(token)); err == nil {
			_, _ = s.db.ExecContext(ctx, `UPDATE api_keys SET last_used_at = ? WHERE id = ?`, time.Now().Unix(), id)
			return true, nil
		}
	}
	return false, rows.Err()
}

// ActiveKeyCount returns number of non-revoked API keys.
func (s *APIKeyService) ActiveKeyCount(ctx context.Context) (int, error) {
	row := s.db.QueryRowContext(ctx, `SELECT COUNT(1) FROM api_keys WHERE revoked = 0`)
	var n int
	if err := row.Scan(&n); err != nil {
		return 0, err
	}
	return n, nil
}

// RevokeAPIKey marks an API key as revoked.
func (s *APIKeyService) RevokeAPIKey(ctx context.Context, id string) error {
	res, err := s.db.ExecContext(ctx, `UPDATE api_keys SET revoked = 1 WHERE id = ?`, id)
	if err != nil {
		return fmt.Errorf("revoke api key: %w", err)
	}
	affected, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("revoke api key rows: %w", err)
	}
	if affected == 0 {
		return fmt.Errorf("api key not found: %s", id)
	}
	return nil
}

// RevokeAPIKeyByToken revokes the active API key matching the provided token.
func (s *APIKeyService) RevokeAPIKeyByToken(ctx context.Context, token string) (string, error) {
	id, err := s.findActiveKeyIDByToken(ctx, token)
	if err != nil {
		return "", err
	}
	if err := s.RevokeAPIKey(ctx, id); err != nil {
		return "", err
	}
	return id, nil
}

// RotateAPIKeyByToken revokes current token key and creates a replacement key.
func (s *APIKeyService) RotateAPIKeyByToken(ctx context.Context, token, description string) (string, string, string, error) {
	oldID, err := s.findActiveKeyIDByToken(ctx, token)
	if err != nil {
		return "", "", "", err
	}
	newID, raw, err := s.CreateAPIKey(description)
	if err != nil {
		return "", "", "", err
	}
	if err := s.RevokeAPIKey(ctx, oldID); err != nil {
		return "", "", "", err
	}
	return newID, raw, oldID, nil
}

func (s *APIKeyService) findActiveKeyIDByToken(ctx context.Context, token string) (string, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT id, key_hash FROM api_keys WHERE revoked = 0`)
	if err != nil {
		return "", err
	}
	defer rows.Close()

	for rows.Next() {
		var id, hash string
		if err := rows.Scan(&id, &hash); err != nil {
			return "", err
		}
		if err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(token)); err == nil {
			return id, nil
		}
	}
	if err := rows.Err(); err != nil {
		return "", err
	}
	return "", fmt.Errorf("api key not found for token")
}

func generateAPIKey() (string, error) {
	buf := make([]byte, 32)
	if _, err := rand.Read(buf); err != nil {
		return "", fmt.Errorf("generate api key: %w", err)
	}
	return base64.RawURLEncoding.EncodeToString(buf), nil
}

