package database

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"

	_ "github.com/mattn/go-sqlite3"
)

// Open creates (or opens) the SQLite database at dbPath.
func Open(dbPath string) (*sql.DB, error) {
	if err := os.MkdirAll(filepath.Dir(dbPath), 0o755); err != nil {
		return nil, fmt.Errorf("create db directory: %w", err)
	}

	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		return nil, fmt.Errorf("open sqlite db: %w", err)
	}
	if _, err := db.Exec(`PRAGMA foreign_keys = ON`); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("enable sqlite foreign keys: %w", err)
	}
	return db, nil
}

// Migrate initializes the database schema.
func Migrate(db *sql.DB) error {
	const schema = `
CREATE TABLE IF NOT EXISTS tasks (
    id TEXT PRIMARY KEY,
    title TEXT NOT NULL,
    description TEXT,
    contexts TEXT NOT NULL,
    project_id TEXT,
    location TEXT,
    status TEXT NOT NULL,
    priority TEXT NOT NULL,
    due_date INTEGER,
    created_at INTEGER NOT NULL,
    completed_at INTEGER,
    tags TEXT NOT NULL,
    notes TEXT,
    recurrence TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS task_subtasks (
    task_id TEXT NOT NULL,
    id TEXT NOT NULL,
    position INTEGER NOT NULL,
    title TEXT NOT NULL,
    description TEXT,
    notes TEXT,
    status TEXT NOT NULL,
    priority TEXT NOT NULL,
    due_date INTEGER,
    location TEXT,
    created_at INTEGER NOT NULL,
    completed_at INTEGER,
    PRIMARY KEY (task_id, id),
    FOREIGN KEY (task_id) REFERENCES tasks(id) ON DELETE CASCADE
);

CREATE TABLE IF NOT EXISTS task_dependencies (
    task_id TEXT NOT NULL,
    depends_on_task_id TEXT NOT NULL,
    PRIMARY KEY (task_id, depends_on_task_id),
    FOREIGN KEY (task_id) REFERENCES tasks(id) ON DELETE CASCADE,
    FOREIGN KEY (depends_on_task_id) REFERENCES tasks(id) ON DELETE CASCADE
);

CREATE TABLE IF NOT EXISTS projects (
    id TEXT PRIMARY KEY,
    name TEXT NOT NULL UNIQUE,
    description TEXT,
    created_at INTEGER NOT NULL
);

CREATE TABLE IF NOT EXISTS auth_sessions (
    id TEXT PRIMARY KEY,
    user_id TEXT NOT NULL,
    access_token TEXT NOT NULL UNIQUE,
    refresh_token TEXT UNIQUE,
    expires_at INTEGER NOT NULL,
    refresh_expires_at INTEGER,
    created_at INTEGER NOT NULL,
    revoked INTEGER NOT NULL DEFAULT 0
);

CREATE TABLE IF NOT EXISTS api_keys (
    id TEXT PRIMARY KEY,
    key_hash TEXT NOT NULL UNIQUE,
    token_digest TEXT UNIQUE,
    description TEXT,
    created_at INTEGER NOT NULL,
    last_used_at INTEGER,
    revoked INTEGER NOT NULL DEFAULT 0
);

CREATE TABLE IF NOT EXISTS auth_events (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    event_type TEXT NOT NULL,
    ip_address TEXT,
    user_agent TEXT,
    timestamp INTEGER NOT NULL,
    metadata TEXT
);

CREATE TABLE IF NOT EXISTS users (
    id TEXT PRIMARY KEY,
    email TEXT UNIQUE NOT NULL,
    password_hash TEXT NOT NULL,
    totp_secret TEXT,
    totp_enabled INTEGER NOT NULL DEFAULT 0,
    last_totp_step INTEGER,
    created_at INTEGER NOT NULL
);

CREATE TABLE IF NOT EXISTS task_notifications (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    task_id TEXT NOT NULL,
    reminder_type TEXT NOT NULL,
    due_at INTEGER NOT NULL,
    sent_at INTEGER,
    delivery_status TEXT NOT NULL,
    error_message TEXT,
    created_at INTEGER NOT NULL,
    UNIQUE (task_id, reminder_type, due_at),
    FOREIGN KEY (task_id) REFERENCES tasks(id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_tasks_status ON tasks(status);
CREATE INDEX IF NOT EXISTS idx_tasks_due_date ON tasks(due_date);
CREATE INDEX IF NOT EXISTS idx_task_dependencies_task_id ON task_dependencies(task_id);
CREATE INDEX IF NOT EXISTS idx_sessions_access_token ON auth_sessions(access_token);
CREATE INDEX IF NOT EXISTS idx_api_keys_hash ON api_keys(key_hash);
CREATE INDEX IF NOT EXISTS idx_task_notifications_task_id ON task_notifications(task_id);
CREATE INDEX IF NOT EXISTS idx_task_notifications_status ON task_notifications(delivery_status);
`

	if _, err := db.Exec(schema); err != nil {
		return fmt.Errorf("run database migrations: %w", err)
	}
	if err := addColumnIfMissing(db, "tasks", "project_id", "TEXT"); err != nil {
		return fmt.Errorf("add tasks.project_id column: %w", err)
	}
	if err := addColumnIfMissing(db, "tasks", "location", "TEXT"); err != nil {
		return fmt.Errorf("add tasks.location column: %w", err)
	}
	if err := addColumnIfMissing(db, "auth_sessions", "refresh_expires_at", "INTEGER"); err != nil {
		return fmt.Errorf("add auth_sessions.refresh_expires_at column: %w", err)
	}
	if err := addColumnIfMissing(db, "api_keys", "token_digest", "TEXT"); err != nil {
		return fmt.Errorf("add api_keys.token_digest column: %w", err)
	}
	if err := addColumnIfMissing(db, "users", "last_totp_step", "INTEGER"); err != nil {
		return fmt.Errorf("add users.last_totp_step column: %w", err)
	}
	if err := addColumnIfMissing(db, "task_subtasks", "id", "TEXT"); err != nil {
		return fmt.Errorf("add task_subtasks.id column: %w", err)
	}
	if err := addColumnIfMissing(db, "task_subtasks", "description", "TEXT"); err != nil {
		return fmt.Errorf("add task_subtasks.description column: %w", err)
	}
	if err := addColumnIfMissing(db, "task_subtasks", "notes", "TEXT"); err != nil {
		return fmt.Errorf("add task_subtasks.notes column: %w", err)
	}
	if err := addColumnIfMissing(db, "task_subtasks", "status", "TEXT"); err != nil {
		return fmt.Errorf("add task_subtasks.status column: %w", err)
	}
	if err := addColumnIfMissing(db, "task_subtasks", "priority", "TEXT"); err != nil {
		return fmt.Errorf("add task_subtasks.priority column: %w", err)
	}
	if err := addColumnIfMissing(db, "task_subtasks", "due_date", "INTEGER"); err != nil {
		return fmt.Errorf("add task_subtasks.due_date column: %w", err)
	}
	if err := addColumnIfMissing(db, "task_subtasks", "location", "TEXT"); err != nil {
		return fmt.Errorf("add task_subtasks.location column: %w", err)
	}
	if err := addColumnIfMissing(db, "task_subtasks", "created_at", "INTEGER"); err != nil {
		return fmt.Errorf("add task_subtasks.created_at column: %w", err)
	}
	if _, err := db.Exec(`UPDATE auth_sessions SET refresh_expires_at = created_at + 2592000 WHERE refresh_expires_at IS NULL`); err != nil {
		return fmt.Errorf("backfill auth_sessions.refresh_expires_at: %w", err)
	}
	if _, err := db.Exec(`UPDATE task_subtasks SET id = printf('%s-%d', task_id, position) WHERE id IS NULL OR id = ''`); err != nil {
		return fmt.Errorf("backfill task_subtasks.id: %w", err)
	}
	if _, err := db.Exec(`UPDATE task_subtasks SET status = CASE WHEN completed_at IS NOT NULL THEN 'done' ELSE 'open' END WHERE status IS NULL OR status = ''`); err != nil {
		return fmt.Errorf("backfill task_subtasks.status: %w", err)
	}
	if _, err := db.Exec(`UPDATE task_subtasks SET priority = 'none' WHERE priority IS NULL OR priority = ''`); err != nil {
		return fmt.Errorf("backfill task_subtasks.priority: %w", err)
	}
	if _, err := db.Exec(`UPDATE task_subtasks SET created_at = strftime('%s','now') WHERE created_at IS NULL`); err != nil {
		return fmt.Errorf("backfill task_subtasks.created_at: %w", err)
	}
	if _, err := db.Exec(`CREATE INDEX IF NOT EXISTS idx_tasks_project_id ON tasks(project_id)`); err != nil {
		return fmt.Errorf("create tasks.project_id index: %w", err)
	}
	if _, err := db.Exec(`CREATE INDEX IF NOT EXISTS idx_task_subtasks_task_position ON task_subtasks(task_id, position)`); err != nil {
		return fmt.Errorf("create task_subtasks task/position index: %w", err)
	}
	if _, err := db.Exec(`CREATE UNIQUE INDEX IF NOT EXISTS idx_api_keys_token_digest ON api_keys(token_digest)`); err != nil {
		return fmt.Errorf("create api_keys.token_digest index: %w", err)
	}
	return nil
}

func addColumnIfMissing(db *sql.DB, tableName, columnName, columnDef string) error {
	rows, err := db.Query("PRAGMA table_info(" + tableName + ")")
	if err != nil {
		return err
	}
	defer rows.Close()

	for rows.Next() {
		var (
			cid       int
			name      string
			colType   string
			notNull   int
			dfltValue sql.NullString
			pk        int
		)
		if err := rows.Scan(&cid, &name, &colType, &notNull, &dfltValue, &pk); err != nil {
			return err
		}
		if name == columnName {
			return nil
		}
	}
	if err := rows.Err(); err != nil {
		return err
	}

	_, err = db.Exec("ALTER TABLE " + tableName + " ADD COLUMN " + columnName + " " + columnDef)
	return err
}
