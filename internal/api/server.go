package api

import (
	"context"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"gtd/internal/api/handlers"
	apimw "gtd/internal/api/middleware"
	"gtd/internal/config"
	"gtd/internal/storage"
)

type denyAllValidator struct{}

func (denyAllValidator) ValidateAPIKey(_ context.Context, _ string) (bool, error) { return false, nil }

// Server hosts the next-generation authenticated API routes.
type Server struct {
	router chi.Router
}

// NewServer builds the API skeleton server with middleware and routes.
func NewServer(
	store storage.Backend,
	cfg *config.ServerConfig,
	validator apimw.APIKeyValidator,
	sessionManager handlers.SessionManager,
	notificationRunner handlers.NotificationRunner,
	notificationStore handlers.NotificationConfigStore,
) *Server {
	if cfg == nil {
		cfg = &config.ServerConfig{APITokenName: "gtd_api_key"}
	}
	if validator == nil {
		validator = denyAllValidator{}
	}

	r := chi.NewRouter()
	r.Use(apimw.Logging)
	r.Use(apimw.SecurityHeaders)
	r.Use(apimw.BodyLimit(1 << 20))

	var keyManager handlers.APIKeyManager
	if km, ok := validator.(handlers.APIKeyManager); ok {
		keyManager = km
	}
	var passwordAuth handlers.PasswordAuthManager
	if pm, ok := validator.(handlers.PasswordAuthManager); ok {
		passwordAuth = pm
	}
	var authEvents handlers.AuthEventLogger
	if em, ok := validator.(handlers.AuthEventLogger); ok {
		authEvents = em
	}
	authHandler := &handlers.AuthHandler{
		APIKeys:         keyManager,
		Sessions:        sessionManager,
		PasswordAuth:    passwordAuth,
		Events:          authEvents,
		TokenCookieName: cfg.APITokenName,
		CSRFCookieName:  cfg.CSRFCookie,
		CookieSecure:    cfg.CookieSecure,
	}
	tasksHandler := &handlers.TasksHandler{Store: store}
	projectsHandler := &handlers.ProjectsHandler{Store: store}
	utilHandler := &handlers.UtilityHandler{Store: store}
	notifyHandler := &handlers.NotificationsHandler{
		Runner: notificationRunner,
		Store:  notificationStore,
	}
	statusHandler := &handlers.StatusHandler{
		StartedAt:       time.Now(),
		APIVersion:      "v1",
		HasAPIKeyAuth:   keyManager != nil,
		HasPasswordAuth: passwordAuth != nil,
		NotifyStore:     notificationStore,
	}

	r.Get("/health", handlers.Health)

	registerAPIRoutes := func(r chi.Router) {
		// Login has stricter limits than general authenticated endpoints.
		r.With(apimw.LoginRateLimit()).Post("/auth/login", authHandler.Login)
		r.With(apimw.SetupMFARateLimit()).Post("/auth/setup-mfa", authHandler.SetupMFA)
		r.With(apimw.VerifyMFARateLimit()).Post("/auth/verify-mfa", authHandler.VerifyMFA)
		r.Post("/auth/refresh", authHandler.Refresh)

		r.Group(func(r chi.Router) {
			r.Use(apimw.AuthRateLimit())
			r.Use(apimw.Auth(validator, cfg.APITokenName))
			r.Use(apimw.CSRF(cfg.CSRFCookie))
			r.Get("/auth/csrf", authHandler.CSRF)
			r.Post("/auth/logout", authHandler.Logout)
			r.Post("/auth/apikey/create", authHandler.CreateAPIKey)
			r.Delete("/auth/apikey/{id}", authHandler.RevokeAPIKey)
			r.Get("/auth/sessions", authHandler.ListSessions)
			r.Delete("/auth/sessions/{id}", authHandler.RevokeSession)
			r.Delete("/auth/sessions/all", authHandler.RevokeAll)

			r.Route("/tasks", tasksHandler.Routes)
			r.Route("/projects", projectsHandler.Routes)

			r.Get("/inbox", utilHandler.Inbox)
			r.Get("/today", utilHandler.Today)
			r.Get("/review", utilHandler.Review)
			r.Get("/search", utilHandler.Search)
			r.Get("/status", statusHandler.Get)

			r.Post("/notifications/run-now", notifyHandler.RunNow)
			r.Get("/notifications/config", notifyHandler.GetConfig)
			r.Put("/notifications/config", notifyHandler.UpdateConfig)
		})
	}

	r.Route("/api", registerAPIRoutes)
	r.Route("/api/v1", registerAPIRoutes)
	r.Handle("/*", newWebHandler("web"))

	return &Server{router: r}
}

// Handler exposes the configured router.
func (s *Server) Handler() http.Handler { return s.router }
