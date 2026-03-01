package parser

import (
	"fmt"
	"strings"
	"time"
)

// ParseDueDate parses human-readable due dates
func ParseDueDate(dateStr string) (*time.Time, error) {
	dateStr = strings.ToLower(strings.TrimSpace(dateStr))

	now := time.Now()
	var dueDate time.Time

	switch dateStr {
	case "today":
		dueDate = now
	case "tomorrow":
		dueDate = now.AddDate(0, 0, 1)
	case "next week":
		dueDate = now.AddDate(0, 0, 7)
	case "next month":
		dueDate = now.AddDate(0, 1, 0)
	default:
		// Try parsing as ISO date (2026-02-15)
		parsed, err := time.Parse("2006-01-02", dateStr)
		if err == nil {
			dueDate = parsed
		} else {
			return nil, fmt.Errorf("unable to parse date: %s", dateStr)
		}
	}

	return &dueDate, nil
}
