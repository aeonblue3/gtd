package storage

import "gtd/internal/models"

// Backend defines the storage contract used by CLI and API layers.
// The current JSON Store and the future SQLite/remote stores implement this.
type Backend interface {
	AddTask(task *models.Task) error
	UpdateTask(task *models.Task) error
	GetTask(id string) (*models.Task, error)
	DeleteTask(id string) error
	GetAllTasks() []*models.Task
	GetTasksByStatus(status models.Status) []*models.Task
	GetTasksByContext(context string) []*models.Task
	GetTasksByContexts(contexts []string) []*models.Task
	GetTasksByPriority(priority models.Priority) []*models.Task
	AddSubtask(taskID, title string) error
	CompleteSubtask(taskID string, index int) error
	AddDependency(taskID, depID string) error
	RemoveDependency(taskID, depID string) error
	CanStartTask(taskID string) (bool, error)
}

