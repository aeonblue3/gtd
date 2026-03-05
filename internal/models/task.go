package models

import (
	"time"
)

// SubtaskStatus represents the state of a subtask.
type SubtaskStatus string

const (
	SubtaskStatusOpen SubtaskStatus = "open"
	SubtaskStatusDone SubtaskStatus = "done"
)

// Subtask represents a nested unit of work under a task.
type Subtask struct {
	ID          string        `json:"id"`
	Title       string        `json:"title"`
	Description string        `json:"description,omitempty"`
	Notes       string        `json:"notes,omitempty"`
	Status      SubtaskStatus `json:"status"`
	Priority    Priority      `json:"priority"`
	DueDate     *time.Time    `json:"dueDate,omitempty"`
	Location    string        `json:"location,omitempty"`
	CreatedAt   time.Time     `json:"createdAt"`
	CompletedAt *time.Time    `json:"completedAt,omitempty"`
}

// Task represents a single task in the GTD system
type Task struct {
	ID          string     `json:"id"`
	Title       string     `json:"title"`
	Description string     `json:"description"`
	Contexts    []string   `json:"context"`
	ProjectID   string     `json:"projectId,omitempty"`
	Location    string     `json:"location,omitempty"`
	Status      Status     `json:"status"`
	Priority    Priority   `json:"priority"`
	DueDate     *time.Time `json:"dueDate,omitempty"`
	CreatedAt   time.Time  `json:"createdAt"`
	CompletedAt *time.Time `json:"completedAt,omitempty"`
	Tags        []string   `json:"tags"`
	Notes       string     `json:"notes"`
	LinkedTasks []string   `json:"linkedTasks"`
	Subtasks    []Subtask  `json:"subtasks"`
	Recurrence  Recurrence `json:"recurrence"`
}

// Project represents a named grouping for tasks.
type Project struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	CreatedAt   time.Time `json:"createdAt"`
}

// Status represents the current state of a task
type Status string

const (
	StatusInbox      Status = "inbox"
	StatusActionable Status = "actionable"
	StatusWaiting    Status = "waiting"
	StatusSomeday    Status = "someday"
	StatusDone       Status = "done"
)

// Priority represents task priority level
type Priority string

const (
	PriorityNone   Priority = "none"
	PriorityLow    Priority = "low"
	PriorityMedium Priority = "medium"
	PriorityHigh   Priority = "high"
)

// Recurrence represents task recurrence pattern
type Recurrence string

const (
	RecurrenceNone    Recurrence = "none"
	RecurrenceDaily   Recurrence = "daily"
	RecurrenceWeekly  Recurrence = "weekly"
	RecurrenceMonthly Recurrence = "monthly"
)

// Config represents application configuration
type Config struct {
	Contexts            []string `json:"contexts"`
	DefaultContext      string   `json:"defaultContext"`
	AutoCommit          bool     `json:"autoCommit"`
	AutoSync            bool     `json:"autoSync"`
	SyncIntervalMinutes int      `json:"syncIntervalMinutes"`
	GitRemote           string   `json:"gitRemote"`
	GitBranch           string   `json:"gitBranch"`
	Mode                string   `json:"mode,omitempty"`
	ServerURL           string   `json:"server_url,omitempty"`
	APIKeyFile          string   `json:"api_key_file,omitempty"`
	LocalDBPath         string   `json:"local_db_path,omitempty"`
	DataPath            string   `json:"-"`
}

// NewTask creates a new task with sensible defaults
func NewTask(title string) *Task {
	return &Task{
		Title:       title,
		Status:      StatusInbox,
		Priority:    PriorityNone,
		Recurrence:  RecurrenceNone,
		CreatedAt:   time.Now(),
		Contexts:    []string{},
		Tags:        []string{},
		LinkedTasks: []string{},
		Subtasks:    []Subtask{},
	}
}

// IsOverdue returns true if task has passed due date and isn't done
func (t *Task) IsOverdue() bool {
	if t.DueDate == nil || t.Status == StatusDone {
		return false
	}
	return t.DueDate.Before(time.Now())
}

// DaysUntilDue returns days until due date (negative if overdue)
func (t *Task) DaysUntilDue() int {
	if t.DueDate == nil {
		return 0
	}
	return int(t.DueDate.Sub(time.Now()).Hours() / 24)
}

// NextRecurringInstance creates the next recurring task instance.
// Returns nil when recurrence is disabled.
func (t *Task) NextRecurringInstance() *Task {
	if t.Recurrence == RecurrenceNone {
		return nil
	}

	next := NewTask(t.Title)
	next.Description = t.Description
	next.Contexts = append([]string{}, t.Contexts...)
	next.ProjectID = t.ProjectID
	next.Location = t.Location
	next.Status = StatusActionable
	next.Priority = t.Priority
	next.Tags = append([]string{}, t.Tags...)
	next.Notes = t.Notes
	next.LinkedTasks = append([]string{}, t.LinkedTasks...)
	next.Recurrence = t.Recurrence
	next.Subtasks = []Subtask{}
	for _, sub := range t.Subtasks {
		next.Subtasks = append(next.Subtasks, Subtask{
			Title:       sub.Title,
			Description: sub.Description,
			Notes:       sub.Notes,
			Status:      SubtaskStatusOpen,
			Priority:    sub.Priority,
			DueDate:     sub.DueDate,
			Location:    sub.Location,
			CreatedAt:   time.Now(),
		})
	}

	base := time.Now()
	if t.DueDate != nil {
		base = *t.DueDate
	}
	switch t.Recurrence {
	case RecurrenceDaily:
		n := base.AddDate(0, 0, 1)
		next.DueDate = &n
	case RecurrenceWeekly:
		n := base.AddDate(0, 0, 7)
		next.DueDate = &n
	case RecurrenceMonthly:
		n := base.AddDate(0, 1, 0)
		next.DueDate = &n
	}

	return next
}
