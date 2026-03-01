package handlers

import (
	"net/http"
	"strings"
	"time"

	"gtd/internal/models"
	"gtd/internal/storage"
)

// UtilityHandler serves convenience query endpoints.
type UtilityHandler struct {
	Store storage.Backend
}

func (h *UtilityHandler) Inbox(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, h.Store.GetTasksByStatus(models.StatusInbox))
}

func (h *UtilityHandler) Today(w http.ResponseWriter, _ *http.Request) {
	actionable := h.Store.GetTasksByStatus(models.StatusActionable)
	now := time.Now()
	out := make([]*models.Task, 0, len(actionable))
	for _, task := range actionable {
		if task.DueDate == nil {
			continue
		}
		y1, m1, d1 := now.Date()
		y2, m2, d2 := task.DueDate.Date()
		if y1 == y2 && m1 == m2 && d1 == d2 {
			out = append(out, task)
		}
	}
	writeJSON(w, http.StatusOK, out)
}

func (h *UtilityHandler) Review(w http.ResponseWriter, _ *http.Request) {
	done := h.Store.GetTasksByStatus(models.StatusDone)
	weekAgo := time.Now().AddDate(0, 0, -7)
	completedThisWeek := 0
	for _, task := range done {
		if task.CompletedAt != nil && task.CompletedAt.After(weekAgo) {
			completedThisWeek++
		}
	}
	writeJSON(w, http.StatusOK, map[string]int{
		"inbox":               len(h.Store.GetTasksByStatus(models.StatusInbox)),
		"actionable":          len(h.Store.GetTasksByStatus(models.StatusActionable)),
		"waiting":             len(h.Store.GetTasksByStatus(models.StatusWaiting)),
		"someday":             len(h.Store.GetTasksByStatus(models.StatusSomeday)),
		"done":                len(done),
		"completed_this_week": completedThisWeek,
	})
}

func (h *UtilityHandler) Search(w http.ResponseWriter, r *http.Request) {
	q := strings.TrimSpace(r.URL.Query().Get("q"))
	if q == "" {
		writeError(w, http.StatusBadRequest, "missing q query parameter")
		return
	}
	q = strings.ToLower(q)
	var out []*models.Task
	for _, task := range h.Store.GetAllTasks() {
		if strings.Contains(strings.ToLower(task.Title), q) ||
			strings.Contains(strings.ToLower(task.Description), q) ||
			strings.Contains(strings.ToLower(task.Notes), q) {
			out = append(out, task)
		}
	}
	writeJSON(w, http.StatusOK, out)
}

