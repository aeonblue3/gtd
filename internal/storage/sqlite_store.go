package storage

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"slices"
	"strings"
	"time"

	"github.com/google/uuid"
	"gtd/internal/database"
	"gtd/internal/models"
)

// SQLiteStore persists tasks in SQLite while preserving current task fields.
type SQLiteStore struct {
	db *sql.DB
}

var _ Backend = (*SQLiteStore)(nil)

// NewSQLiteStore opens and migrates a SQLite-backed storage instance.
func NewSQLiteStore(dbPath string) (*SQLiteStore, error) {
	db, err := database.Open(dbPath)
	if err != nil {
		return nil, err
	}
	if err := database.Migrate(db); err != nil {
		_ = db.Close()
		return nil, err
	}
	return &SQLiteStore{db: db}, nil
}

// Close releases the underlying database handle.
func (s *SQLiteStore) Close() error {
	return s.db.Close()
}

func (s *SQLiteStore) AddTask(task *models.Task) error {
	if task.ID == "" {
		task.ID = uuid.New().String()
	}
	if task.CreatedAt.IsZero() {
		task.CreatedAt = time.Now()
	}
	if task.Recurrence == "" {
		task.Recurrence = models.RecurrenceNone
	}
	task.Location = strings.TrimSpace(task.Location)
	task.ProjectID = strings.TrimSpace(task.ProjectID)
	if task.ProjectID != "" && !s.projectExists(task.ProjectID) {
		return fmt.Errorf("project not found: %s", task.ProjectID)
	}

	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	if err := s.upsertTaskRow(tx, task, false); err != nil {
		return err
	}
	if err := s.replaceSubtasks(tx, task.ID, task.Subtasks); err != nil {
		return err
	}
	if err := s.replaceDependencies(tx, task.ID, task.LinkedTasks); err != nil {
		return err
	}
	return tx.Commit()
}

func (s *SQLiteStore) UpdateTask(task *models.Task) error {
	task.Location = strings.TrimSpace(task.Location)
	task.ProjectID = strings.TrimSpace(task.ProjectID)
	if task.ProjectID != "" && !s.projectExists(task.ProjectID) {
		return fmt.Errorf("project not found: %s", task.ProjectID)
	}

	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	if err := s.upsertTaskRow(tx, task, true); err != nil {
		return err
	}
	if err := s.replaceSubtasks(tx, task.ID, task.Subtasks); err != nil {
		return err
	}
	if err := s.replaceDependencies(tx, task.ID, task.LinkedTasks); err != nil {
		return err
	}
	return tx.Commit()
}

func (s *SQLiteStore) GetTask(id string) (*models.Task, error) {
	row := s.db.QueryRow(`
SELECT id, title, description, contexts, project_id, location, status, priority, due_date, created_at, completed_at, tags, notes, recurrence
FROM tasks
WHERE id = ?`, id)

	task, err := scanTask(row)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("task not found: %s", id)
		}
		return nil, err
	}

	subtasks, err := s.loadSubtasks(id)
	if err != nil {
		return nil, err
	}
	task.Subtasks = subtasks

	deps, err := s.loadDependencies(id)
	if err != nil {
		return nil, err
	}
	task.LinkedTasks = deps

	return task, nil
}

func (s *SQLiteStore) DeleteTask(id string) error {
	res, err := s.db.Exec(`DELETE FROM tasks WHERE id = ?`, id)
	if err != nil {
		return err
	}
	n, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if n == 0 {
		return fmt.Errorf("task not found: %s", id)
	}
	return nil
}

func (s *SQLiteStore) GetAllTasks() []*models.Task {
	rows, err := s.db.Query(`SELECT id FROM tasks`)
	if err != nil {
		return []*models.Task{}
	}
	defer rows.Close()

	var out []*models.Task
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			continue
		}
		task, err := s.GetTask(id)
		if err != nil {
			continue
		}
		out = append(out, task)
	}
	return out
}

func (s *SQLiteStore) GetTasksByStatus(status models.Status) []*models.Task {
	rows, err := s.db.Query(`SELECT id FROM tasks WHERE status = ?`, string(status))
	if err != nil {
		return []*models.Task{}
	}
	defer rows.Close()

	var out []*models.Task
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			continue
		}
		task, err := s.GetTask(id)
		if err != nil {
			continue
		}
		out = append(out, task)
	}
	return out
}

func (s *SQLiteStore) GetTasksByContext(context string) []*models.Task {
	var out []*models.Task
	for _, task := range s.GetAllTasks() {
		for _, ctx := range task.Contexts {
			if ctx == context {
				out = append(out, task)
				break
			}
		}
	}
	return out
}

func (s *SQLiteStore) GetTasksByContexts(contexts []string) []*models.Task {
	if len(contexts) == 0 {
		return s.GetAllTasks()
	}
	contextMap := map[string]bool{}
	for _, ctx := range contexts {
		contextMap[ctx] = true
	}
	var out []*models.Task
	for _, task := range s.GetAllTasks() {
		for _, ctx := range task.Contexts {
			if contextMap[ctx] {
				out = append(out, task)
				break
			}
		}
	}
	return out
}

func (s *SQLiteStore) GetTasksByPriority(priority models.Priority) []*models.Task {
	rows, err := s.db.Query(`SELECT id FROM tasks WHERE priority = ?`, string(priority))
	if err != nil {
		return []*models.Task{}
	}
	defer rows.Close()

	var out []*models.Task
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			continue
		}
		task, err := s.GetTask(id)
		if err != nil {
			continue
		}
		out = append(out, task)
	}
	return out
}

func (s *SQLiteStore) AddSubtask(taskID, title string) error {
	task, err := s.GetTask(taskID)
	if err != nil {
		return err
	}
	task.Subtasks = append(task.Subtasks, models.Subtask{
		ID:        uuid.NewString(),
		Title:     strings.TrimSpace(title),
		Status:    models.SubtaskStatusOpen,
		Priority:  models.PriorityNone,
		CreatedAt: time.Now(),
	})
	return s.UpdateTask(task)
}

func (s *SQLiteStore) CompleteSubtask(taskID string, index int) error {
	task, err := s.GetTask(taskID)
	if err != nil {
		return err
	}
	if index < 0 || index >= len(task.Subtasks) {
		return fmt.Errorf("subtask index out of range")
	}
	now := time.Now()
	task.Subtasks[index].Status = models.SubtaskStatusDone
	task.Subtasks[index].CompletedAt = &now
	return s.UpdateTask(task)
}

func (s *SQLiteStore) AddDependency(taskID, depID string) error {
	if taskID == depID {
		return fmt.Errorf("task cannot depend on itself")
	}
	if !s.taskExists(taskID) {
		return fmt.Errorf("task not found: %s", taskID)
	}
	if !s.taskExists(depID) {
		return fmt.Errorf("dependency task not found: %s", depID)
	}

	graph, err := s.dependencyGraph()
	if err != nil {
		return err
	}
	if createsCycleGraph(graph, taskID, depID) {
		return fmt.Errorf("dependency would create a cycle")
	}

	_, err = s.db.Exec(`INSERT OR IGNORE INTO task_dependencies (task_id, depends_on_task_id) VALUES (?, ?)`, taskID, depID)
	return err
}

func (s *SQLiteStore) RemoveDependency(taskID, depID string) error {
	_, err := s.db.Exec(`DELETE FROM task_dependencies WHERE task_id = ? AND depends_on_task_id = ?`, taskID, depID)
	return err
}

func (s *SQLiteStore) CanStartTask(taskID string) (bool, error) {
	rows, err := s.db.Query(`
SELECT t.status
FROM task_dependencies d
JOIN tasks t ON t.id = d.depends_on_task_id
WHERE d.task_id = ?`, taskID)
	if err != nil {
		return false, err
	}
	defer rows.Close()

	for rows.Next() {
		var status string
		if err := rows.Scan(&status); err != nil {
			return false, err
		}
		if models.Status(status) != models.StatusDone {
			return false, nil
		}
	}
	return true, rows.Err()
}

func (s *SQLiteStore) CreateProject(project *models.Project) error {
	name := strings.TrimSpace(project.Name)
	if name == "" {
		return fmt.Errorf("project name is required")
	}
	if project.ID == "" {
		project.ID = uuid.New().String()
	}
	if project.CreatedAt.IsZero() {
		project.CreatedAt = time.Now()
	}
	project.Name = name
	project.Description = strings.TrimSpace(project.Description)

	_, err := s.db.Exec(
		`INSERT INTO projects (id, name, description, created_at) VALUES (?, ?, ?, ?)`,
		project.ID, project.Name, project.Description, project.CreatedAt.Unix(),
	)
	return err
}

func (s *SQLiteStore) UpdateProject(project *models.Project) error {
	name := strings.TrimSpace(project.Name)
	if name == "" {
		return fmt.Errorf("project name is required")
	}
	project.Name = name
	project.Description = strings.TrimSpace(project.Description)

	res, err := s.db.Exec(
		`UPDATE projects SET name = ?, description = ? WHERE id = ?`,
		project.Name, project.Description, project.ID,
	)
	if err != nil {
		return err
	}
	affected, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if affected == 0 {
		return fmt.Errorf("project not found: %s", project.ID)
	}
	return nil
}

func (s *SQLiteStore) GetProject(id string) (*models.Project, error) {
	row := s.db.QueryRow(`SELECT id, name, description, created_at FROM projects WHERE id = ?`, id)
	var (
		project       models.Project
		description   sql.NullString
		createdAtUnix int64
	)
	if err := row.Scan(&project.ID, &project.Name, &description, &createdAtUnix); err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("project not found: %s", id)
		}
		return nil, err
	}
	project.Description = strings.TrimSpace(description.String)
	project.CreatedAt = time.Unix(createdAtUnix, 0)
	return &project, nil
}

func (s *SQLiteStore) DeleteProject(id string) error {
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	res, err := tx.Exec(`DELETE FROM projects WHERE id = ?`, id)
	if err != nil {
		return err
	}
	affected, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if affected == 0 {
		return fmt.Errorf("project not found: %s", id)
	}
	if _, err := tx.Exec(`UPDATE tasks SET project_id = NULL WHERE project_id = ?`, id); err != nil {
		return err
	}
	return tx.Commit()
}

func (s *SQLiteStore) GetAllProjects() []*models.Project {
	rows, err := s.db.Query(`SELECT id, name, description, created_at FROM projects ORDER BY created_at DESC, name ASC`)
	if err != nil {
		return []*models.Project{}
	}
	defer rows.Close()

	out := make([]*models.Project, 0)
	for rows.Next() {
		var (
			project       models.Project
			description   sql.NullString
			createdAtUnix int64
		)
		if err := rows.Scan(&project.ID, &project.Name, &description, &createdAtUnix); err != nil {
			continue
		}
		project.Description = strings.TrimSpace(description.String)
		project.CreatedAt = time.Unix(createdAtUnix, 0)
		out = append(out, &project)
	}
	return out
}

func (s *SQLiteStore) upsertTaskRow(tx *sql.Tx, task *models.Task, update bool) error {
	contextsJSON, err := json.Marshal(task.Contexts)
	if err != nil {
		return err
	}
	tagsJSON, err := json.Marshal(task.Tags)
	if err != nil {
		return err
	}

	if update {
		res, err := tx.Exec(`
UPDATE tasks
SET title = ?, description = ?, contexts = ?, project_id = ?, location = ?, status = ?, priority = ?, due_date = ?, created_at = ?, completed_at = ?, tags = ?, notes = ?, recurrence = ?
WHERE id = ?`,
			task.Title, task.Description, string(contextsJSON), nullIfEmpty(task.ProjectID), nullIfEmpty(task.Location), string(task.Status), string(task.Priority),
			timeToUnix(task.DueDate), task.CreatedAt.Unix(), timeToUnix(task.CompletedAt), string(tagsJSON), task.Notes, string(task.Recurrence), task.ID)
		if err != nil {
			return err
		}
		affected, err := res.RowsAffected()
		if err != nil {
			return err
		}
		if affected == 0 {
			return fmt.Errorf("task not found: %s", task.ID)
		}
		return nil
	}

	_, err = tx.Exec(`
INSERT INTO tasks (id, title, description, contexts, project_id, location, status, priority, due_date, created_at, completed_at, tags, notes, recurrence)
VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		task.ID, task.Title, task.Description, string(contextsJSON), nullIfEmpty(task.ProjectID), nullIfEmpty(task.Location), string(task.Status), string(task.Priority),
		timeToUnix(task.DueDate), task.CreatedAt.Unix(), timeToUnix(task.CompletedAt), string(tagsJSON), task.Notes, string(task.Recurrence))
	return err
}

func (s *SQLiteStore) replaceSubtasks(tx *sql.Tx, taskID string, subtasks []models.Subtask) error {
	if _, err := tx.Exec(`DELETE FROM task_subtasks WHERE task_id = ?`, taskID); err != nil {
		return err
	}
	for i, subtask := range subtasks {
		if strings.TrimSpace(subtask.ID) == "" {
			subtask.ID = uuid.NewString()
		}
		if subtask.CreatedAt.IsZero() {
			subtask.CreatedAt = time.Now()
		}
		if subtask.Status == "" {
			subtask.Status = models.SubtaskStatusOpen
		}
		if subtask.Priority == "" {
			subtask.Priority = models.PriorityNone
		}
		if subtask.Status == models.SubtaskStatusDone && subtask.CompletedAt == nil {
			now := time.Now()
			subtask.CompletedAt = &now
		}
		if subtask.Status == models.SubtaskStatusOpen {
			subtask.CompletedAt = nil
		}
		if _, err := tx.Exec(`
INSERT INTO task_subtasks (task_id, id, position, title, description, notes, status, priority, due_date, location, created_at, completed_at)
VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
			taskID, subtask.ID, i, subtask.Title, strings.TrimSpace(subtask.Description), strings.TrimSpace(subtask.Notes),
			string(subtask.Status), string(subtask.Priority), timeToUnix(subtask.DueDate), nullIfEmpty(subtask.Location),
			subtask.CreatedAt.Unix(), timeToUnix(subtask.CompletedAt)); err != nil {
			return err
		}
	}
	return nil
}

func (s *SQLiteStore) replaceDependencies(tx *sql.Tx, taskID string, deps []string) error {
	if _, err := tx.Exec(`DELETE FROM task_dependencies WHERE task_id = ?`, taskID); err != nil {
		return err
	}
	seen := map[string]bool{}
	for _, depID := range deps {
		if depID == taskID || seen[depID] {
			continue
		}
		seen[depID] = true
		if _, err := tx.Exec(`
INSERT OR IGNORE INTO task_dependencies (task_id, depends_on_task_id)
VALUES (?, ?)`, taskID, depID); err != nil {
			return err
		}
	}
	return nil
}

func (s *SQLiteStore) loadSubtasks(taskID string) ([]models.Subtask, error) {
	rows, err := s.db.Query(`
SELECT id, title, description, notes, status, priority, due_date, location, created_at, completed_at
FROM task_subtasks
WHERE task_id = ?
ORDER BY position ASC`, taskID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var subtasks []models.Subtask
	for rows.Next() {
		var (
			sub                    models.Subtask
			status, priority       string
			dueDate, completedAt   sql.NullInt64
			location               sql.NullString
			createdAtUnix          int64
		)
		if err := rows.Scan(
			&sub.ID, &sub.Title, &sub.Description, &sub.Notes, &status, &priority, &dueDate, &location, &createdAtUnix, &completedAt,
		); err != nil {
			return nil, err
		}
		sub.Status = models.SubtaskStatus(status)
		sub.Priority = models.Priority(priority)
		sub.Location = strings.TrimSpace(location.String)
		sub.CreatedAt = time.Unix(createdAtUnix, 0)
		if dueDate.Valid {
			d := time.Unix(dueDate.Int64, 0)
			sub.DueDate = &d
		}
		if completedAt.Valid {
			t := time.Unix(completedAt.Int64, 0)
			sub.CompletedAt = &t
		}
		subtasks = append(subtasks, sub)
	}
	return subtasks, rows.Err()
}

func (s *SQLiteStore) loadDependencies(taskID string) ([]string, error) {
	rows, err := s.db.Query(`
SELECT depends_on_task_id
FROM task_dependencies
WHERE task_id = ?
ORDER BY depends_on_task_id ASC`, taskID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var deps []string
	for rows.Next() {
		var depID string
		if err := rows.Scan(&depID); err != nil {
			return nil, err
		}
		deps = append(deps, depID)
	}
	return deps, rows.Err()
}

func (s *SQLiteStore) dependencyGraph() (map[string][]string, error) {
	rows, err := s.db.Query(`SELECT task_id, depends_on_task_id FROM task_dependencies`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	graph := map[string][]string{}
	for rows.Next() {
		var taskID, depID string
		if err := rows.Scan(&taskID, &depID); err != nil {
			return nil, err
		}
		graph[taskID] = append(graph[taskID], depID)
	}
	return graph, rows.Err()
}

func (s *SQLiteStore) taskExists(taskID string) bool {
	row := s.db.QueryRow(`SELECT 1 FROM tasks WHERE id = ? LIMIT 1`, taskID)
	var one int
	return row.Scan(&one) == nil
}

func (s *SQLiteStore) projectExists(projectID string) bool {
	row := s.db.QueryRow(`SELECT 1 FROM projects WHERE id = ? LIMIT 1`, projectID)
	var one int
	return row.Scan(&one) == nil
}

func createsCycleGraph(graph map[string][]string, taskID, depID string) bool {
	g := map[string][]string{}
	for k, v := range graph {
		g[k] = append([]string{}, v...)
	}
	if !slices.Contains(g[taskID], depID) {
		g[taskID] = append(g[taskID], depID)
	}

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
		for _, nxt := range g[cur] {
			if dfs(nxt) {
				return true
			}
		}
		return false
	}
	return dfs(depID)
}

func scanTask(scanner interface {
	Scan(dest ...any) error
}) (*models.Task, error) {
	var (
		task                   models.Task
		contextsJSON, tagsJSON string
		dueDate, completedAt   sql.NullInt64
		recurrence, status     string
		priority               string
		projectID, location    sql.NullString
		createdAtUnix          int64
	)
	if err := scanner.Scan(
		&task.ID, &task.Title, &task.Description, &contextsJSON, &projectID, &location, &status, &priority,
		&dueDate, &createdAtUnix, &completedAt, &tagsJSON, &task.Notes, &recurrence,
	); err != nil {
		return nil, err
	}

	task.Status = models.Status(status)
	task.Priority = models.Priority(priority)
	task.Recurrence = models.Recurrence(recurrence)
	task.CreatedAt = time.Unix(createdAtUnix, 0)
	task.Contexts = []string{}
	task.Tags = []string{}
	task.Subtasks = []models.Subtask{}
	task.LinkedTasks = []string{}
	task.ProjectID = strings.TrimSpace(projectID.String)
	task.Location = strings.TrimSpace(location.String)

	if err := json.Unmarshal([]byte(contextsJSON), &task.Contexts); err != nil {
		return nil, err
	}
	if err := json.Unmarshal([]byte(tagsJSON), &task.Tags); err != nil {
		return nil, err
	}
	if dueDate.Valid {
		t := time.Unix(dueDate.Int64, 0)
		task.DueDate = &t
	}
	if completedAt.Valid {
		t := time.Unix(completedAt.Int64, 0)
		task.CompletedAt = &t
	}
	return &task, nil
}

func timeToUnix(t *time.Time) any {
	if t == nil {
		return nil
	}
	return t.Unix()
}

func nullIfEmpty(v string) any {
	s := strings.TrimSpace(v)
	if s == "" {
		return nil
	}
	return s
}
