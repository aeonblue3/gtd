package handlers

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
	"gtd/internal/models"
	"gtd/internal/storage"
)

// ProjectsHandler serves CRUD endpoints for task projects.
type ProjectsHandler struct {
	Store storage.Backend
}

// Routes mounts project routes in the current router.
func (h *ProjectsHandler) Routes(r chi.Router) {
	r.Get("/", h.list)
	r.Post("/", h.create)
	r.Get("/{id}", h.get)
	r.Put("/{id}", h.update)
	r.Delete("/{id}", h.delete)
}

func (h *ProjectsHandler) list(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, h.Store.GetAllProjects())
}

func (h *ProjectsHandler) create(w http.ResponseWriter, r *http.Request) {
	var in projectInput
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		writeError(w, http.StatusBadRequest, "invalid json body")
		return
	}
	name := strings.TrimSpace(in.Name)
	if name == "" {
		writeError(w, http.StatusBadRequest, "name is required")
		return
	}
	project := &models.Project{
		Name:        name,
		Description: strings.TrimSpace(in.Description),
	}
	if err := h.Store.CreateProject(project); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, project)
}

func (h *ProjectsHandler) get(w http.ResponseWriter, r *http.Request) {
	project, err := h.Store.GetProject(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, http.StatusNotFound, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, project)
}

func (h *ProjectsHandler) update(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	project, err := h.Store.GetProject(id)
	if err != nil {
		writeError(w, http.StatusNotFound, err.Error())
		return
	}

	var in projectInput
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		writeError(w, http.StatusBadRequest, "invalid json body")
		return
	}
	name := strings.TrimSpace(in.Name)
	if name == "" {
		writeError(w, http.StatusBadRequest, "name is required")
		return
	}

	project.Name = name
	project.Description = strings.TrimSpace(in.Description)
	if err := h.Store.UpdateProject(project); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, project)
}

func (h *ProjectsHandler) delete(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if err := h.Store.DeleteProject(id); err != nil {
		writeError(w, http.StatusNotFound, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"deleted": id})
}

type projectInput struct {
	Name        string `json:"name"`
	Description string `json:"description"`
}
