package handlers

import (
	"fmt"
	"strings"

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

