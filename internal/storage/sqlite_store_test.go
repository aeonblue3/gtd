package storage

import (
	"path/filepath"
	"testing"
	"time"

	"gtd/internal/models"
)

func newTestSQLiteStore(t *testing.T) *SQLiteStore {
	t.Helper()
	s, err := NewSQLiteStore(filepath.Join(t.TempDir(), "gtd.db"))
	if err != nil {
		t.Fatalf("failed to create sqlite store: %v", err)
	}
	t.Cleanup(func() {
		_ = s.Close()
	})
	return s
}

func TestSQLiteStoreRoundTripAllTaskFields(t *testing.T) {
	s := newTestSQLiteStore(t)

	due := time.Now().Add(48 * time.Hour).Truncate(time.Second)
	doneAt := time.Now().Add(-2 * time.Hour).Truncate(time.Second)
	other := models.NewTask("Dependency")
	if err := s.AddTask(other); err != nil {
		t.Fatal(err)
	}

	project := &models.Project{
		Name:        "Platform",
		Description: "Core platform improvements",
	}
	if err := s.CreateProject(project); err != nil {
		t.Fatal(err)
	}

	task := &models.Task{
		Title:       "Ship phase 0",
		Description: "storage abstraction and sqlite backend",
		Contexts:    []string{"work", "deep"},
		ProjectID:   project.ID,
		Location:    "Home Office",
		Status:      models.StatusActionable,
		Priority:    models.PriorityHigh,
		DueDate:     &due,
		CreatedAt:   time.Now().Add(-24 * time.Hour).Truncate(time.Second),
		CompletedAt: &doneAt,
		Tags:        []string{"gtd", "migration"},
		Notes:       "Keep CLI behavior unchanged",
		LinkedTasks: []string{other.ID},
		Subtasks: []models.Subtask{
			{Title: "design"},
			{Title: "implement", CompletedAt: &doneAt},
		},
		Recurrence: models.RecurrenceWeekly,
	}

	if err := s.AddTask(task); err != nil {
		t.Fatal(err)
	}

	got, err := s.GetTask(task.ID)
	if err != nil {
		t.Fatal(err)
	}
	if got.Title != task.Title || got.Description != task.Description || got.Notes != task.Notes {
		t.Fatalf("task scalar fields mismatch: got=%+v", got)
	}
	if got.ProjectID != project.ID {
		t.Fatalf("project mismatch: got=%q want=%q", got.ProjectID, project.ID)
	}
	if got.Location != "Home Office" {
		t.Fatalf("location mismatch: got=%q", got.Location)
	}
	if len(got.Contexts) != 2 || got.Contexts[0] != "work" || got.Contexts[1] != "deep" {
		t.Fatalf("contexts mismatch: %#v", got.Contexts)
	}
	if len(got.Tags) != 2 || got.Tags[0] != "gtd" || got.Tags[1] != "migration" {
		t.Fatalf("tags mismatch: %#v", got.Tags)
	}
	if got.DueDate == nil || got.DueDate.Unix() != due.Unix() {
		t.Fatalf("due date mismatch: got=%v want=%v", got.DueDate, due)
	}
	if got.CompletedAt == nil || got.CompletedAt.Unix() != doneAt.Unix() {
		t.Fatalf("completed date mismatch: got=%v want=%v", got.CompletedAt, doneAt)
	}
	if got.Recurrence != models.RecurrenceWeekly {
		t.Fatalf("recurrence mismatch: got=%s", got.Recurrence)
	}
	if len(got.Subtasks) != 2 || got.Subtasks[1].CompletedAt == nil {
		t.Fatalf("subtasks mismatch: %#v", got.Subtasks)
	}
	if len(got.LinkedTasks) != 1 || got.LinkedTasks[0] != other.ID {
		t.Fatalf("dependencies mismatch: %#v", got.LinkedTasks)
	}
}

func TestSQLiteProjectCRUD(t *testing.T) {
	s := newTestSQLiteStore(t)

	project := &models.Project{Name: "Operations", Description: "Ops tracking"}
	if err := s.CreateProject(project); err != nil {
		t.Fatal(err)
	}

	got, err := s.GetProject(project.ID)
	if err != nil {
		t.Fatal(err)
	}
	if got.Name != "Operations" {
		t.Fatalf("unexpected name: %q", got.Name)
	}

	project.Name = "Operations + Infra"
	if err := s.UpdateProject(project); err != nil {
		t.Fatal(err)
	}
	projects := s.GetAllProjects()
	if len(projects) != 1 || projects[0].Name != "Operations + Infra" {
		t.Fatalf("unexpected projects: %#v", projects)
	}

	task := models.NewTask("Task with project")
	task.ProjectID = project.ID
	if err := s.AddTask(task); err != nil {
		t.Fatal(err)
	}

	if err := s.DeleteProject(project.ID); err != nil {
		t.Fatal(err)
	}
	updated, err := s.GetTask(task.ID)
	if err != nil {
		t.Fatal(err)
	}
	if updated.ProjectID != "" {
		t.Fatalf("expected task project cleared, got %q", updated.ProjectID)
	}
}

func TestSQLiteDependencyCycleDetection(t *testing.T) {
	s := newTestSQLiteStore(t)
	a := models.NewTask("A")
	b := models.NewTask("B")
	c := models.NewTask("C")
	if err := s.AddTask(a); err != nil {
		t.Fatal(err)
	}
	if err := s.AddTask(b); err != nil {
		t.Fatal(err)
	}
	if err := s.AddTask(c); err != nil {
		t.Fatal(err)
	}
	if err := s.AddDependency(a.ID, b.ID); err != nil {
		t.Fatal(err)
	}
	if err := s.AddDependency(b.ID, c.ID); err != nil {
		t.Fatal(err)
	}
	if err := s.AddDependency(c.ID, a.ID); err == nil {
		t.Fatal("expected cycle detection error")
	}
}
