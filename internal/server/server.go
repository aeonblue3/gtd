package server

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"gtd/internal/models"
	"gtd/internal/storage"
)

// Server hosts a simple HTTP API for tasks.
type Server struct {
	Store storage.Backend
}

// New returns a configured HTTP server wrapper.
func New(store storage.Backend) *Server {
	return &Server{Store: store}
}

// Handler returns the API mux.
func (s *Server) Handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/healthz", s.handleHealth)
	mux.HandleFunc("/tasks", s.handleTasks)
	mux.HandleFunc("/tasks/", s.handleTaskByID)
	mux.HandleFunc("/webhooks/remarkable/complete", s.handleRemarkableComplete)
	return mux
}

func (s *Server) handleHealth(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (s *Server) handleTasks(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		writeJSON(w, http.StatusOK, s.Store.GetAllTasks())
	case http.MethodPost:
		var in struct {
			Title    string            `json:"title"`
			Contexts []string          `json:"contexts"`
			Priority models.Priority   `json:"priority"`
			Status   models.Status     `json:"status"`
			DueDate  *time.Time        `json:"dueDate"`
			Tags     []string          `json:"tags"`
			Rec      models.Recurrence `json:"recurrence"`
		}
		if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
			writeErr(w, http.StatusBadRequest, err)
			return
		}
		if strings.TrimSpace(in.Title) == "" {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "title required"})
			return
		}
		task := models.NewTask(in.Title)
		task.Contexts = in.Contexts
		task.Priority = in.Priority
		task.Status = in.Status
		task.DueDate = in.DueDate
		task.Tags = in.Tags
		if in.Rec != "" {
			task.Recurrence = in.Rec
		}
		if task.Status == "" {
			task.Status = models.StatusInbox
		}
		if task.Priority == "" {
			task.Priority = models.PriorityNone
		}
		if err := s.Store.AddTask(task); err != nil {
			writeErr(w, http.StatusInternalServerError, err)
			return
		}
		writeJSON(w, http.StatusCreated, task)
	default:
		w.WriteHeader(http.StatusMethodNotAllowed)
	}
}

func (s *Server) handleTaskByID(w http.ResponseWriter, r *http.Request) {
	id := strings.TrimPrefix(r.URL.Path, "/tasks/")
	if id == "" {
		w.WriteHeader(http.StatusNotFound)
		return
	}
	task, err := s.Store.GetTask(id)
	if err != nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": err.Error()})
		return
	}
	switch r.Method {
	case http.MethodGet:
		writeJSON(w, http.StatusOK, task)
	case http.MethodPatch:
		var in map[string]any
		if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
			writeErr(w, http.StatusBadRequest, err)
			return
		}
		if v, ok := in["title"].(string); ok && strings.TrimSpace(v) != "" {
			task.Title = v
		}
		if v, ok := in["status"].(string); ok {
			task.Status = models.Status(v)
			if task.Status == models.StatusDone {
				now := time.Now()
				task.CompletedAt = &now
			}
		}
		if err := s.Store.UpdateTask(task); err != nil {
			writeErr(w, http.StatusInternalServerError, err)
			return
		}
		writeJSON(w, http.StatusOK, task)
	case http.MethodDelete:
		if err := s.Store.DeleteTask(task.ID); err != nil {
			writeErr(w, http.StatusInternalServerError, err)
			return
		}
		writeJSON(w, http.StatusOK, map[string]string{"deleted": task.ID})
	default:
		w.WriteHeader(http.StatusMethodNotAllowed)
	}
}

func (s *Server) handleRemarkableComplete(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	var in struct {
		TaskID string `json:"taskId"`
		Title  string `json:"title"`
	}
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		writeErr(w, http.StatusBadRequest, err)
		return
	}
	var target *models.Task
	if in.TaskID != "" {
		t, err := s.Store.GetTask(in.TaskID)
		if err != nil {
			writeErr(w, http.StatusNotFound, err)
			return
		}
		target = t
	} else {
		for _, t := range s.Store.GetAllTasks() {
			if strings.EqualFold(t.Title, in.Title) {
				target = t
				break
			}
		}
	}
	if target == nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "task not found"})
		return
	}
	now := time.Now()
	target.Status = models.StatusDone
	target.CompletedAt = &now
	if err := s.Store.UpdateTask(target); err != nil {
		writeErr(w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"completed": target.ID})
}

func writeErr(w http.ResponseWriter, code int, err error) {
	writeJSON(w, code, map[string]string{"error": err.Error()})
}

func writeJSON(w http.ResponseWriter, code int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	if err := json.NewEncoder(w).Encode(v); err != nil {
		http.Error(w, fmt.Sprintf(`{"error":"%s"}`, err.Error()), http.StatusInternalServerError)
	}
}
