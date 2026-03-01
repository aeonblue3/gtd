package notify

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"gtd/internal/database"
)

func TestRunOnceCreatesDryRunNotification(t *testing.T) {
	db, err := database.Open(filepath.Join(t.TempDir(), "gtd.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if err := database.Migrate(db); err != nil {
		t.Fatal(err)
	}

	now := time.Unix(1_700_000_000, 0)
	_, err = db.Exec(`
INSERT INTO tasks (id, title, description, contexts, status, priority, due_date, created_at, tags, notes, recurrence)
VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		"t-1", "Pay bill", "", "[]", "actionable", "high", now.Add(30*time.Minute).Unix(), now.Unix(), "[]", "", "none")
	if err != nil {
		t.Fatal(err)
	}

	svc, err := NewService(db, Config{
		Enabled:    true,
		CheckEvery: time.Minute,
		DryRun:     true,
		EmailTo:    "me@example.com",
		EmailFrom:  "gtd@example.com",
	}, nil)
	if err != nil {
		t.Fatal(err)
	}
	svc.now = func() time.Time { return now }

	stats, err := svc.RunOnce(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if stats.DryRun != 1 {
		t.Fatalf("expected dry_run count 1, got %#v", stats)
	}

	var (
		status string
		count  int
	)
	row := db.QueryRow(`SELECT delivery_status FROM task_notifications WHERE task_id = ?`, "t-1")
	if err := row.Scan(&status); err != nil {
		t.Fatal(err)
	}
	if status != "dry_run" {
		t.Fatalf("expected dry_run status, got %s", status)
	}
	row = db.QueryRow(`SELECT COUNT(1) FROM task_notifications WHERE task_id = ?`, "t-1")
	if err := row.Scan(&count); err != nil {
		t.Fatal(err)
	}
	if count != 1 {
		t.Fatalf("expected one notification row, got %d", count)
	}
}

func TestRunOnceIsDeduplicated(t *testing.T) {
	db, err := database.Open(filepath.Join(t.TempDir(), "gtd.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if err := database.Migrate(db); err != nil {
		t.Fatal(err)
	}

	now := time.Unix(1_700_100_000, 0)
	_, err = db.Exec(`
INSERT INTO tasks (id, title, description, contexts, status, priority, due_date, created_at, tags, notes, recurrence)
VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		"t-2", "Finish report", "", "[]", "actionable", "medium", now.Add(2*time.Hour).Unix(), now.Unix(), "[]", "", "none")
	if err != nil {
		t.Fatal(err)
	}

	svc, err := NewService(db, Config{
		Enabled:    true,
		CheckEvery: time.Minute,
		DryRun:     true,
		EmailTo:    "me@example.com",
		EmailFrom:  "gtd@example.com",
	}, nil)
	if err != nil {
		t.Fatal(err)
	}
	svc.now = func() time.Time { return now }

	if _, err := svc.RunOnce(context.Background()); err != nil {
		t.Fatal(err)
	}
	if _, err := svc.RunOnce(context.Background()); err != nil {
		t.Fatal(err)
	}

	var count int
	row := db.QueryRow(`SELECT COUNT(1) FROM task_notifications WHERE task_id = ?`, "t-2")
	if err := row.Scan(&count); err != nil {
		t.Fatal(err)
	}
	if count != 1 {
		t.Fatalf("expected dedupe to keep one row, got %d", count)
	}
}

