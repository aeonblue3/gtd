package storage

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"time"

	"github.com/google/uuid"
	"gtd/internal/models"
)

// Store handles reading and writing tasks to disk
type Store struct {
	tasksDir string
	tasks    map[string]*models.Task
}

var _ Backend = (*Store)(nil)

// NewStore creates a new storage instance
func NewStore(dataPath string) (*Store, error) {
	tasksDir := filepath.Join(dataPath, "tasks")

	// Create directories if they don't exist
	if err := os.MkdirAll(tasksDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create tasks directory: %w", err)
	}

	s := &Store{
		tasksDir: tasksDir,
		tasks:    make(map[string]*models.Task),
	}

	// Load existing tasks
	if err := s.loadAll(); err != nil {
		return nil, fmt.Errorf("failed to load tasks: %w", err)
	}

	return s, nil
}

// loadAll loads all tasks from disk into memory
func (s *Store) loadAll() error {
	files, err := ioutil.ReadDir(s.tasksDir)
	if err != nil {
		return err
	}

	for _, file := range files {
		if !file.IsDir() && strings.HasSuffix(file.Name(), ".json") {
			path := filepath.Join(s.tasksDir, file.Name())
			task, err := s.loadTask(path)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Warning: failed to load %s: %v\n", file.Name(), err)
				continue
			}
			s.tasks[task.ID] = task
		}
	}

	return nil
}

// loadTask loads a single task from a JSON file
func (s *Store) loadTask(path string) (*models.Task, error) {
	data, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var task models.Task
	if err := json.Unmarshal(data, &task); err != nil {
		return nil, err
	}

	return &task, nil
}

// AddTask creates and saves a new task
func (s *Store) AddTask(task *models.Task) error {
	if task.ID == "" {
		task.ID = uuid.New().String()
	}

	s.tasks[task.ID] = task
	return s.saveTask(task)
}

// UpdateTask updates an existing task
func (s *Store) UpdateTask(task *models.Task) error {
	if _, exists := s.tasks[task.ID]; !exists {
		return fmt.Errorf("task not found: %s", task.ID)
	}

	s.tasks[task.ID] = task
	return s.saveTask(task)
}

// saveTask persists a task to disk
func (s *Store) saveTask(task *models.Task) error {
	data, err := json.MarshalIndent(task, "", "  ")
	if err != nil {
		return err
	}

	path := filepath.Join(s.tasksDir, task.ID+".json")
	if err := ioutil.WriteFile(path, data, 0644); err != nil {
		return err
	}

	return nil
}

// GetTask retrieves a task by ID
func (s *Store) GetTask(id string) (*models.Task, error) {
	task, exists := s.tasks[id]
	if !exists {
		return nil, fmt.Errorf("task not found: %s", id)
	}
	return task, nil
}

// DeleteTask removes a task
func (s *Store) DeleteTask(id string) error {
	if _, exists := s.tasks[id]; !exists {
		return fmt.Errorf("task not found: %s", id)
	}

	path := filepath.Join(s.tasksDir, id+".json")
	if err := os.Remove(path); err != nil {
		return err
	}

	delete(s.tasks, id)
	return nil
}

// GetAllTasks returns all tasks
func (s *Store) GetAllTasks() []*models.Task {
	tasks := make([]*models.Task, 0, len(s.tasks))
	for _, task := range s.tasks {
		tasks = append(tasks, task)
	}
	return tasks
}

// GetTasksByStatus returns tasks filtered by status
func (s *Store) GetTasksByStatus(status models.Status) []*models.Task {
	var result []*models.Task
	for _, task := range s.tasks {
		if task.Status == status {
			result = append(result, task)
		}
	}
	return result
}

// GetTasksByContext returns tasks filtered by context
func (s *Store) GetTasksByContext(context string) []*models.Task {
	var result []*models.Task
	for _, task := range s.tasks {
		for _, ctx := range task.Contexts {
			if ctx == context {
				result = append(result, task)
				break
			}
		}
	}
	return result
}

// GetTasksByContexts returns tasks that match any of the given contexts
func (s *Store) GetTasksByContexts(contexts []string) []*models.Task {
	if len(contexts) == 0 {
		return s.GetAllTasks()
	}

	contextMap := make(map[string]bool)
	for _, c := range contexts {
		contextMap[c] = true
	}

	var result []*models.Task
	for _, task := range s.tasks {
		for _, ctx := range task.Contexts {
			if contextMap[ctx] {
				result = append(result, task)
				break
			}
		}
	}
	return result
}

// GetTasksByPriority returns tasks filtered by priority
func (s *Store) GetTasksByPriority(priority models.Priority) []*models.Task {
	var result []*models.Task
	for _, task := range s.tasks {
		if task.Priority == priority {
			result = append(result, task)
		}
	}
	return result
}

// AddSubtask adds a subtask to a task.
func (s *Store) AddSubtask(taskID, title string) error {
	task, err := s.GetTask(taskID)
	if err != nil {
		return err
	}
	task.Subtasks = append(task.Subtasks, models.Subtask{Title: strings.TrimSpace(title)})
	return s.UpdateTask(task)
}

// CompleteSubtask marks a subtask complete by index.
func (s *Store) CompleteSubtask(taskID string, index int) error {
	task, err := s.GetTask(taskID)
	if err != nil {
		return err
	}
	if index < 0 || index >= len(task.Subtasks) {
		return fmt.Errorf("subtask index out of range")
	}
	now := time.Now()
	task.Subtasks[index].CompletedAt = &now
	return s.UpdateTask(task)
}

// AddDependency links taskID to depID, preventing cycles.
func (s *Store) AddDependency(taskID, depID string) error {
	task, err := s.GetTask(taskID)
	if err != nil {
		return err
	}
	if _, err := s.GetTask(depID); err != nil {
		return fmt.Errorf("dependency task not found: %s", depID)
	}
	if taskID == depID {
		return fmt.Errorf("task cannot depend on itself")
	}
	if createsCycle(s.tasks, taskID, depID) {
		return fmt.Errorf("dependency would create a cycle")
	}
	if !slices.Contains(task.LinkedTasks, depID) {
		task.LinkedTasks = append(task.LinkedTasks, depID)
	}
	return s.UpdateTask(task)
}

// RemoveDependency removes a linked dependency.
func (s *Store) RemoveDependency(taskID, depID string) error {
	task, err := s.GetTask(taskID)
	if err != nil {
		return err
	}
	out := make([]string, 0, len(task.LinkedTasks))
	for _, id := range task.LinkedTasks {
		if id != depID {
			out = append(out, id)
		}
	}
	task.LinkedTasks = out
	return s.UpdateTask(task)
}

// CanStartTask checks whether all dependencies are done.
func (s *Store) CanStartTask(taskID string) (bool, error) {
	task, err := s.GetTask(taskID)
	if err != nil {
		return false, err
	}
	for _, dep := range task.LinkedTasks {
		depTask, err := s.GetTask(dep)
		if err != nil {
			return false, err
		}
		if depTask.Status != models.StatusDone {
			return false, nil
		}
	}
	return true, nil
}

func createsCycle(tasks map[string]*models.Task, taskID, depID string) bool {
	visited := map[string]bool{}
	var dfs func(string) bool
	dfs = func(cur string) bool {
		if cur == taskID {
			return true
		}
		if visited[cur] {
			return false
		}
		visited[cur] = true
		task := tasks[cur]
		if task == nil {
			return false
		}
		for _, nxt := range task.LinkedTasks {
			if dfs(nxt) {
				return true
			}
		}
		return false
	}
	return dfs(depID)
}
