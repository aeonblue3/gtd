package handlers

import (
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"gtd/internal/models"
)

func normalizeAndValidateStatus(raw string) (models.Status, error) {
	value := models.Status(strings.TrimSpace(raw))
	switch value {
	case models.StatusInbox, models.StatusActionable, models.StatusWaiting, models.StatusSomeday, models.StatusDone:
		return value, nil
	default:
		return "", fmt.Errorf("invalid status: %s", raw)
	}
}

func normalizeAndValidatePriority(raw string) (models.Priority, error) {
	value := models.Priority(strings.TrimSpace(raw))
	switch value {
	case models.PriorityNone, models.PriorityLow, models.PriorityMedium, models.PriorityHigh:
		return value, nil
	default:
		return "", fmt.Errorf("invalid priority: %s", raw)
	}
}

func normalizeAndValidateRecurrence(raw string) (models.Recurrence, error) {
	value := models.Recurrence(strings.TrimSpace(raw))
	switch value {
	case models.RecurrenceNone, models.RecurrenceDaily, models.RecurrenceWeekly, models.RecurrenceMonthly:
		return value, nil
	default:
		return "", fmt.Errorf("invalid recurrence: %s", raw)
	}
}

func cleanStringSlice(values []string) []string {
	out := make([]string, 0, len(values))
	seen := map[string]bool{}
	for _, value := range values {
		item := strings.TrimSpace(value)
		if item == "" || seen[item] {
			continue
		}
		seen[item] = true
		out = append(out, item)
	}
	return out
}

func normalizeAndValidateSubtasks(values []models.Subtask) ([]models.Subtask, error) {
	out := make([]models.Subtask, 0, len(values))
	seenIDs := map[string]bool{}
	for _, raw := range values {
		title := strings.TrimSpace(raw.Title)
		if title == "" {
			return nil, fmt.Errorf("subtask title is required")
		}
		id := strings.TrimSpace(raw.ID)
		if id == "" {
			id = uuid.NewString()
		}
		if seenIDs[id] {
			return nil, fmt.Errorf("duplicate subtask id: %s", id)
		}
		seenIDs[id] = true

		status := models.SubtaskStatus(strings.TrimSpace(string(raw.Status)))
		if status == "" {
			status = models.SubtaskStatusOpen
		}
		switch status {
		case models.SubtaskStatusOpen, models.SubtaskStatusDone:
		default:
			return nil, fmt.Errorf("invalid subtask status: %s", raw.Status)
		}

		priority := raw.Priority
		if strings.TrimSpace(string(priority)) == "" {
			priority = models.PriorityNone
		} else {
			normalizedPriority, err := normalizeAndValidatePriority(string(priority))
			if err != nil {
				return nil, fmt.Errorf("invalid subtask priority: %s", raw.Priority)
			}
			priority = normalizedPriority
		}

		createdAt := raw.CreatedAt
		if createdAt.IsZero() {
			createdAt = time.Now()
		}
		completedAt := raw.CompletedAt
		if status == models.SubtaskStatusDone && completedAt == nil {
			now := time.Now()
			completedAt = &now
		}
		if status == models.SubtaskStatusOpen {
			completedAt = nil
		}

		out = append(out, models.Subtask{
			ID:          id,
			Title:       title,
			Description: strings.TrimSpace(raw.Description),
			Notes:       strings.TrimSpace(raw.Notes),
			Status:      status,
			Priority:    priority,
			DueDate:     raw.DueDate,
			Location:    strings.TrimSpace(raw.Location),
			CreatedAt:   createdAt,
			CompletedAt: completedAt,
		})
	}
	return out, nil
}

