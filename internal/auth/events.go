package auth

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"
)

// EventService writes auth/security events to auth_events.
type EventService struct {
	db *sql.DB
}

// NewEventService creates an auth event service.
func NewEventService(db *sql.DB) *EventService {
	return &EventService{db: db}
}

// Record writes an event with optional metadata.
func (s *EventService) Record(ctx context.Context, eventType, ipAddress, userAgent string, metadata map[string]any) error {
	var metaJSON string
	if metadata != nil {
		raw, err := json.Marshal(metadata)
		if err != nil {
			return fmt.Errorf("marshal auth event metadata: %w", err)
		}
		metaJSON = string(raw)
	}
	_, err := s.db.ExecContext(ctx, `
INSERT INTO auth_events (event_type, ip_address, user_agent, timestamp, metadata)
VALUES (?, ?, ?, ?, ?)`,
		eventType, ipAddress, userAgent, time.Now().Unix(), metaJSON)
	if err != nil {
		return fmt.Errorf("insert auth event: %w", err)
	}
	return nil
}

