package models

import "testing"

func TestNextRecurringInstance(t *testing.T) {
	task := NewTask("Recurring")
	task.Recurrence = RecurrenceWeekly
	task.ProjectID = "p-1"
	task.Location = "Home Office"
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
	if next.ProjectID != "p-1" {
		t.Fatalf("expected project copy, got %q", next.ProjectID)
	}
	if next.Location != "Home Office" {
		t.Fatalf("expected location copy, got %q", next.Location)
	}
}
