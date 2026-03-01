package notify

import (
	"context"
	"crypto/tls"
	"database/sql"
	"fmt"
	"log"
	"net/smtp"
	"strings"
	"sync"
	"time"

	"gtd/internal/config"
	"gtd/internal/models"
)

const (
	reminderDue24h  = "due_24h"
	reminderDue1h   = "due_1h"
	reminderOverdue = "overdue"
)

// Sender delivers notification emails.
type Sender interface {
	Send(to, subject, body string) error
}

// Config controls scheduler and delivery behavior.
type Config struct {
	Enabled      bool
	CheckEvery   time.Duration
	DryRun       bool
	EmailTo      string
	EmailFrom    string
	SMTPHost     string
	SMTPPort     int
	SMTPUser     string
	SMTPPassword string
}

// FromServerConfig maps server config to notification config.
func FromServerConfig(cfg *config.ServerConfig) Config {
	return Config{
		Enabled:      cfg.NotificationsEnabled,
		CheckEvery:   time.Duration(cfg.NotificationCheckSeconds) * time.Second,
		DryRun:       cfg.NotificationDryRun,
		EmailTo:      strings.TrimSpace(cfg.NotificationEmailTo),
		EmailFrom:    strings.TrimSpace(cfg.NotificationEmailFrom),
		SMTPHost:     strings.TrimSpace(cfg.NotificationSMTPHost),
		SMTPPort:     cfg.NotificationSMTPPort,
		SMTPUser:     strings.TrimSpace(cfg.NotificationSMTPUsername),
		SMTPPassword: cfg.NotificationSMTPPassword,
	}
}

// Service scans for due tasks and sends notifications.
type Service struct {
	db     *sql.DB
	cfg    Config
	sender Sender
	logger *log.Logger
	now    func() time.Time
	mu     sync.RWMutex
}

// RunStats summarizes a single reminder scan pass.
type RunStats struct {
	Scanned   int `json:"scanned"`
	Attempted int `json:"attempted"`
	Sent      int `json:"sent"`
	DryRun    int `json:"dry_run"`
	Failed    int `json:"failed"`
	Skipped   int `json:"skipped"`
}

// NewService builds a notification service.
func NewService(db *sql.DB, cfg Config, logger *log.Logger) (*Service, error) {
	if logger == nil {
		logger = log.Default()
	}
	if cfg.CheckEvery <= 0 {
		cfg.CheckEvery = 5 * time.Minute
	}
	if cfg.EmailFrom == "" {
		cfg.EmailFrom = "gtd@localhost"
	}
	var sender Sender = logSender{logger: logger}
	if !cfg.DryRun {
		if cfg.EmailTo == "" || cfg.SMTPHost == "" || cfg.SMTPPort <= 0 {
			return nil, fmt.Errorf("notifications enabled without complete smtp configuration")
		}
		sender = smtpSender{
			host:     cfg.SMTPHost,
			port:     cfg.SMTPPort,
			username: cfg.SMTPUser,
			password: cfg.SMTPPassword,
			from:     cfg.EmailFrom,
		}
	}
	return &Service{
		db:     db,
		cfg:    cfg,
		sender: sender,
		logger: logger,
		now:    time.Now,
	}, nil
}

// Start runs the periodic scheduler loop until context cancellation.
func (s *Service) Start(ctx context.Context) {
	_, _ = s.RunOnce(ctx)
	ticker := time.NewTicker(s.checkEvery())
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			_, _ = s.RunOnce(ctx)
		}
	}
}

// RunOnce scans for reminder candidates and attempts delivery.
func (s *Service) RunOnce(ctx context.Context) (RunStats, error) {
	candidates, err := s.findCandidates(ctx)
	if err != nil {
		return RunStats{}, err
	}
	stats := RunStats{Scanned: len(candidates)}
	for _, c := range candidates {
		result, err := s.processCandidate(ctx, c)
		if result != "skipped" {
			stats.Attempted++
		}
		switch result {
		case "sent":
			stats.Sent++
		case "dry_run":
			stats.DryRun++
		case "failed":
			stats.Failed++
		case "skipped":
			stats.Skipped++
		}
		if err != nil {
			s.logger.Printf("notification candidate failed task=%s type=%s: %v", c.TaskID, c.Type, err)
		}
	}
	return stats, nil
}

type candidate struct {
	TaskID string
	Title  string
	DueAt  int64
	Type   string
}

func (s *Service) findCandidates(ctx context.Context) ([]candidate, error) {
	now := s.now().Unix()
	plus24h := s.now().Add(24 * time.Hour).Unix()

	rows, err := s.db.QueryContext(ctx, `
SELECT id, title, due_date
FROM tasks
WHERE due_date IS NOT NULL AND status != ? AND due_date <= ?`, string(models.StatusDone), plus24h)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := []candidate{}
	for rows.Next() {
		var id, title string
		var dueAt int64
		if err := rows.Scan(&id, &title, &dueAt); err != nil {
			return nil, err
		}
		kind := reminderDue24h
		if dueAt <= now {
			kind = reminderOverdue
		} else if dueAt <= s.now().Add(time.Hour).Unix() {
			kind = reminderDue1h
		}
		out = append(out, candidate{TaskID: id, Title: title, DueAt: dueAt, Type: kind})
	}
	return out, rows.Err()
}

func (s *Service) processCandidate(ctx context.Context, c candidate) (string, error) {
	res, err := s.db.ExecContext(ctx, `
INSERT OR IGNORE INTO task_notifications (task_id, reminder_type, due_at, delivery_status, created_at)
VALUES (?, ?, ?, ?, ?)`,
		c.TaskID, c.Type, c.DueAt, "pending", s.now().Unix())
	if err != nil {
		return "failed", err
	}
	affected, err := res.RowsAffected()
	if err != nil {
		return "failed", err
	}
	if affected == 0 {
		return "skipped", nil
	}

	subject := buildSubject(c)
	body := buildBody(c)
	status := "sent"
	cfg := s.snapshotConfig()
	if cfg.DryRun {
		status = "dry_run"
		s.logger.Printf("[dry-run] notify %s | %s", cfg.EmailTo, subject)
	} else {
		sender := s.snapshotSender()
		if err := sender.Send(cfg.EmailTo, subject, body); err != nil {
			_, _ = s.db.ExecContext(ctx, `
UPDATE task_notifications
SET delivery_status = ?, error_message = ?, sent_at = ?
WHERE task_id = ? AND reminder_type = ? AND due_at = ?`,
				"failed", err.Error(), s.now().Unix(), c.TaskID, c.Type, c.DueAt)
			return "failed", err
		}
	}

	_, err = s.db.ExecContext(ctx, `
UPDATE task_notifications
SET delivery_status = ?, sent_at = ?, error_message = ''
WHERE task_id = ? AND reminder_type = ? AND due_at = ?`,
		status, s.now().Unix(), c.TaskID, c.Type, c.DueAt)
	if err != nil {
		return "failed", err
	}
	return status, nil
}

func buildSubject(c candidate) string {
	switch c.Type {
	case reminderOverdue:
		return fmt.Sprintf("[GTD] Overdue: %s", c.Title)
	case reminderDue1h:
		return fmt.Sprintf("[GTD] Due in 1 hour: %s", c.Title)
	default:
		return fmt.Sprintf("[GTD] Due within 24 hours: %s", c.Title)
	}
}

func buildBody(c candidate) string {
	due := time.Unix(c.DueAt, 0).Format(time.RFC1123)
	return fmt.Sprintf("Task: %s\nTask ID: %s\nDue: %s\nReminder: %s\n", c.Title, c.TaskID, due, c.Type)
}

type logSender struct {
	logger *log.Logger
}

func (s logSender) Send(to, subject, body string) error {
	s.logger.Printf("[notify] to=%s subject=%q body=%q", to, subject, body)
	return nil
}

type smtpSender struct {
	host     string
	port     int
	username string
	password string
	from     string
}

func (s smtpSender) Send(to, subject, body string) error {
	addr := fmt.Sprintf("%s:%d", s.host, s.port)
	msg := strings.Builder{}
	msg.WriteString(fmt.Sprintf("From: %s\r\n", s.from))
	msg.WriteString(fmt.Sprintf("To: %s\r\n", to))
	msg.WriteString(fmt.Sprintf("Subject: %s\r\n", subject))
	msg.WriteString("MIME-Version: 1.0\r\n")
	msg.WriteString("Content-Type: text/plain; charset=\"utf-8\"\r\n")
	msg.WriteString("\r\n")
	msg.WriteString(body)

	tlsCfg := &tls.Config{
		ServerName: s.host,
		MinVersion: tls.VersionTLS12,
	}
	var (
		client *smtp.Client
		err    error
	)
	if s.port == 465 {
		conn, dialErr := tls.Dial("tcp", addr, tlsCfg)
		if dialErr != nil {
			return dialErr
		}
		client, err = smtp.NewClient(conn, s.host)
	} else {
		client, err = smtp.Dial(addr)
		if err == nil {
			if ok, _ := client.Extension("STARTTLS"); !ok {
				_ = client.Close()
				return fmt.Errorf("smtp server does not support STARTTLS")
			}
			if err = client.StartTLS(tlsCfg); err != nil {
				_ = client.Close()
				return fmt.Errorf("starttls failed: %w", err)
			}
		}
	}
	if err != nil {
		return err
	}
	defer client.Close()

	if s.username != "" {
		auth := smtp.PlainAuth("", s.username, s.password, s.host)
		if err := client.Auth(auth); err != nil {
			return err
		}
	}
	if err := client.Mail(s.from); err != nil {
		return err
	}
	if err := client.Rcpt(to); err != nil {
		return err
	}
	w, err := client.Data()
	if err != nil {
		return err
	}
	if _, err := w.Write([]byte(msg.String())); err != nil {
		return err
	}
	if err := w.Close(); err != nil {
		return err
	}
	return client.Quit()
}

// SnapshotConfig returns the currently active notification config.
func (s *Service) SnapshotConfig() Config {
	return s.snapshotConfig()
}

// UpdateConfig applies a new notification config at runtime.
func (s *Service) UpdateConfig(cfg Config) error {
	if cfg.CheckEvery <= 0 {
		cfg.CheckEvery = 5 * time.Minute
	}
	if cfg.EmailFrom == "" {
		cfg.EmailFrom = "gtd@localhost"
	}
	var sender Sender = logSender{logger: s.logger}
	if !cfg.DryRun {
		if cfg.EmailTo == "" || cfg.SMTPHost == "" || cfg.SMTPPort <= 0 {
			return fmt.Errorf("notifications enabled without complete smtp configuration")
		}
		sender = smtpSender{
			host:     cfg.SMTPHost,
			port:     cfg.SMTPPort,
			username: cfg.SMTPUser,
			password: cfg.SMTPPassword,
			from:     cfg.EmailFrom,
		}
	}
	s.mu.Lock()
	s.cfg = cfg
	s.sender = sender
	s.mu.Unlock()
	return nil
}

func (s *Service) snapshotConfig() Config {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.cfg
}

func (s *Service) snapshotSender() Sender {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.sender
}

func (s *Service) checkEvery() time.Duration {
	cfg := s.snapshotConfig()
	if cfg.CheckEvery <= 0 {
		return 5 * time.Minute
	}
	return cfg.CheckEvery
}
