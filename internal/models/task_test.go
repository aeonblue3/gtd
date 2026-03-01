package models

import "testing"

func TestNextRecurringInstance(t *testing.T) {
	task := NewTask("Recurring")
	task.Recurrence = RecurrenceWeekly
	next := task.NextRecurringInstance()
	if next == nil {
		t.Fatal("expected recurring instance")
	}
	if next.Recurrence != RecurrenceWeekly {
		t.Fatalf("unexpected recurrence: %s", next.Recurrence)
	}
	if next.Status != StatusActionable {
		t.Fatalf("expected actionable status, got %s", next.Status)
	}
	if next.DueDate == nil {
		t.Fatal("expected due date")
	}
}
