package handlers

import (
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"gtd/internal/models"
	"gtd/internal/storage"
)

// TasksHandler serves task CRUD endpoints.
type TasksHandler struct {
	Store storage.Backend
}

// Routes mounts task routes in the current router.
func (h *TasksHandler) Routes(r chi.Router) {
	r.Get("/", h.list)
	r.Post("/", h.create)
	r.Get("/{id}", h.get)
	r.Put("/{id}", h.update)
	r.Delete("/{id}", h.delete)
	r.Post("/{id}/complete", h.complete)
}

func (h *TasksHandler) list(w http.ResponseWriter, r *http.Request) {
	tasks := h.Store.GetAllTasks()

	if status := strings.TrimSpace(r.URL.Query().Get("status")); status != "" {
		normalized, err := normalizeAndValidateStatus(status)
		if err != nil {
			writeError(w, http.StatusBadRequest, err.Error())
			return
		}
		tasks = filterByStatus(tasks, normalized)
	}
	if context := strings.TrimSpace(r.URL.Query().Get("context")); context != "" {
		tasks = filterByContext(tasks, context)
	}
	if priority := strings.TrimSpace(r.URL.Query().Get("priority")); priority != "" {
		normalized, err := normalizeAndValidatePriority(priority)
		if err != nil {
			writeError(w, http.StatusBadRequest, err.Error())
			return
		}
		tasks = filterByPriority(tasks, normalized)
	}

	writeJSON(w, http.StatusOK, tasks)
}

func (h *TasksHandler) create(w http.ResponseWriter, r *http.Request) {
	var in createTaskInput
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		writeError(w, http.StatusBadRequest, "invalid json body")
		return
	}
	title := strings.TrimSpace(in.Title)
	if title == "" {
		writeError(w, http.StatusBadRequest, "title is required")
		return
	}
	task := models.NewTask(title)
	task.Description = strings.TrimSpace(in.Description)
	task.Contexts = cleanStringSlice(in.Contexts())
	task.DueDate = in.DueDate
	task.Tags = cleanStringSlice(in.Tags)
	task.Notes = strings.TrimSpace(in.Notes)
	task.LinkedTasks = cleanStringSlice(in.LinkedTasks)
	task.Subtasks = in.Subtasks

	if strings.TrimSpace(in.Status) == "" {
		task.Status = models.StatusInbox
	} else {
		status, err := normalizeAndValidateStatus(in.Status)
		if err != nil {
			writeError(w, http.StatusBadRequest, err.Error())
			return
		}
		task.Status = status
	}

	if strings.TrimSpace(in.Priority) == "" {
		task.Priority = models.PriorityNone
	} else {
		priority, err := normalizeAndValidatePriority(in.Priority)
		if err != nil {
			writeError(w, http.StatusBadRequest, err.Error())
			return
		}
		task.Priority = priority
	}

	if strings.TrimSpace(in.Recurrence) == "" {
		task.Recurrence = models.RecurrenceNone
	} else {
		recurrence, err := normalizeAndValidateRecurrence(in.Recurrence)
		if err != nil {
			writeError(w, http.StatusBadRequest, err.Error())
			return
		}
		task.Recurrence = recurrence
	}

	if err := h.Store.AddTask(task); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, task)
}

func (h *TasksHandler) get(w http.ResponseWriter, r *http.Request) {
	task, err := h.Store.GetTask(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, http.StatusNotFound, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, task)
}

func (h *TasksHandler) update(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	task, err := h.Store.GetTask(id)
	if err != nil {
		writeError(w, http.StatusNotFound, err.Error())
		return
	}

	var in updateTaskInput
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		writeError(w, http.StatusBadRequest, "invalid json body")
		return
	}
	if in.Title != nil {
		title := strings.TrimSpace(*in.Title)
		if title == "" {
			writeError(w, http.StatusBadRequest, "title cannot be empty")
			return
		}
		task.Title = title
	}
	if in.Description != nil {
		task.Description = strings.TrimSpace(*in.Description)
	}
	if in.Contexts != nil {
		task.Contexts = cleanStringSlice(*in.Contexts)
	}
	if in.ContextAlt != nil {
		task.Contexts = cleanStringSlice(*in.ContextAlt)
	}
	if in.Status != nil {
		status, err := normalizeAndValidateStatus(*in.Status)
		if err != nil {
			writeError(w, http.StatusBadRequest, err.Error())
			return
		}
		task.Status = status
		if status == models.StatusDone {
			now := time.Now()
			task.CompletedAt = &now
		}
	}
	if in.Priority != nil {
		priority, err := normalizeAndValidatePriority(*in.Priority)
		if err != nil {
			writeError(w, http.StatusBadRequest, err.Error())
			return
		}
		task.Priority = priority
	}
	if in.ClearDueDate != nil && *in.ClearDueDate {
		task.DueDate = nil
	} else if in.DueDate != nil {
		task.DueDate = in.DueDate
	}
	if in.Tags != nil {
		task.Tags = cleanStringSlice(*in.Tags)
	}
	if in.Notes != nil {
		task.Notes = strings.TrimSpace(*in.Notes)
	}
	if in.LinkedTasks != nil {
		task.LinkedTasks = cleanStringSlice(*in.LinkedTasks)
	}
	if in.Subtasks != nil {
		task.Subtasks = *in.Subtasks
	}
	if in.Recurrence != nil {
		recurrence, err := normalizeAndValidateRecurrence(*in.Recurrence)
		if err != nil {
			writeError(w, http.StatusBadRequest, err.Error())
			return
		}
		task.Recurrence = recurrence
	}

	if err := h.Store.UpdateTask(task); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, task)
}

func (h *TasksHandler) delete(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if err := h.Store.DeleteTask(id); err != nil {
		writeError(w, http.StatusNotFound, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"deleted": id})
}

func (h *TasksHandler) complete(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	task, err := h.Store.GetTask(id)
	if err != nil {
		writeError(w, http.StatusNotFound, err.Error())
		return
	}
	now := time.Now()
	task.Status = models.StatusDone
	task.CompletedAt = &now
	if err := h.Store.UpdateTask(task); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, task)
}

func filterByStatus(tasks []*models.Task, status models.Status) []*models.Task {
	out := make([]*models.Task, 0, len(tasks))
	for _, t := range tasks {
		if t.Status == status {
			out = append(out, t)
		}
	}
	return out
}

func filterByContext(tasks []*models.Task, context string) []*models.Task {
	out := make([]*models.Task, 0, len(tasks))
	for _, t := range tasks {
		for _, c := range t.Contexts {
			if c == context {
				out = append(out, t)
				break
			}
		}
	}
	return out
}

func filterByPriority(tasks []*models.Task, priority models.Priority) []*models.Task {
	out := make([]*models.Task, 0, len(tasks))
	for _, t := range tasks {
		if t.Priority == priority {
			out = append(out, t)
		}
	}
	return out
}

type createTaskInput struct {
	Title        string           `json:"title"`
	Description  string           `json:"description"`
	ContextsJSON []string         `json:"contexts"`
	ContextJSON  []string         `json:"context"`
	Status       string           `json:"status"`
	Priority     string           `json:"priority"`
	DueDate      *time.Time       `json:"dueDate"`
	Tags         []string         `json:"tags"`
	Notes        string           `json:"notes"`
	LinkedTasks  []string         `json:"linkedTasks"`
	Subtasks     []models.Subtask `json:"subtasks"`
	Recurrence   string           `json:"recurrence"`
}

func (c createTaskInput) Contexts() []string {
	if len(c.ContextsJSON) > 0 {
		return c.ContextsJSON
	}
	return c.ContextJSON
}

type updateTaskInput struct {
	Title       *string           `json:"title"`
	Description *string           `json:"description"`
	Contexts    *[]string         `json:"contexts"`
	ContextAlt  *[]string         `json:"context"`
	Status      *string           `json:"status"`
	Priority    *string           `json:"priority"`
	DueDate     *time.Time        `json:"dueDate"`
	ClearDueDate *bool            `json:"clearDueDate"`
	Tags        *[]string         `json:"tags"`
	Notes       *string           `json:"notes"`
	LinkedTasks *[]string         `json:"linkedTasks"`
	Subtasks    *[]models.Subtask `json:"subtasks"`
	Recurrence  *string           `json:"recurrence"`
}

