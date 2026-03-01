package storage

import (
	"testing"

	"gtd/internal/models"
)

func newTestStore(t *testing.T) *Store {
	t.Helper()
	s, err := NewStore(t.TempDir())
	if err != nil {
		t.Fatalf("failed to create store: %v", err)
	}
	return s
}

func TestDependenciesCycleDetection(t *testing.T) {
	s := newTestStore(t)
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

func TestSubtasks(t *testing.T) {
	s := newTestStore(t)
	task := models.NewTask("Task")
	if err := s.AddTask(task); err != nil {
		t.Fatal(err)
	}
	if err := s.AddSubtask(task.ID, "one"); err != nil {
		t.Fatal(err)
	}
	if err := s.CompleteSubtask(task.ID, 0); err != nil {
		t.Fatal(err)
	}
	got, err := s.GetTask(task.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(got.Subtasks) != 1 || got.Subtasks[0].CompletedAt == nil {
		t.Fatal("subtask completion not persisted")
	}
}
