package auth

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"gtd/internal/database"
)

func TestSessionServicePrunesExpiredSessions(t *testing.T) {
	db, err := database.Open(filepath.Join(t.TempDir(), "gtd.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if err := database.Migrate(db); err != nil {
		t.Fatal(err)
	}

	ctx := context.Background()
	_, err = db.ExecContext(ctx, `
INSERT INTO auth_sessions (id, user_id, access_token, refresh_token, expires_at, created_at, revoked)
VALUES (?, ?, ?, ?, ?, ?, 0)`,
		"expired-1", "u-1", "a-expired", "r-expired", time.Now().Add(-time.Minute).Unix(), time.Now().Add(-2*time.Minute).Unix())
	if err != nil {
		t.Fatal(err)
	}

	svc := NewSessionService(db)
	ok, err := svc.ValidateAccessToken(ctx, "a-expired")
	if err != nil {
		t.Fatal(err)
	}
	if ok {
		t.Fatal("expected expired access token to be invalid")
	}

	var count int
	row := db.QueryRowContext(ctx, `SELECT COUNT(1) FROM auth_sessions WHERE id = ?`, "expired-1")
	if err := row.Scan(&count); err != nil {
		t.Fatal(err)
	}
	if count != 0 {
		t.Fatalf("expected expired session to be pruned, found count=%d", count)
	}
}

