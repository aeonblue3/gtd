package handlers

import (
	"net/http"
	"time"
)

// StatusHandler reports service/runtime readiness metadata.
type StatusHandler struct {
	StartedAt       time.Time
	APIVersion      string
	HasAPIKeyAuth   bool
	HasPasswordAuth bool
	NotifyStore     NotificationConfigStore
}

func (h *StatusHandler) Get(w http.ResponseWriter, _ *http.Request) {
	if h.APIVersion == "" {
		h.APIVersion = "v1"
	}
	if h.StartedAt.IsZero() {
		h.StartedAt = time.Now()
	}

	notify := map[string]any{
		"enabled":          false,
		"scheduler_active": false,
		"dry_run":          false,
		"restart_required": false,
	}
	if h.NotifyStore != nil {
		cfg := h.NotifyStore.GetNotificationConfig()
		notify["enabled"] = cfg.Enabled
		notify["scheduler_active"] = cfg.Enabled && !cfg.RestartRequired
		notify["dry_run"] = cfg.DryRun
		notify["restart_required"] = cfg.RestartRequired
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"service": "gtd-api",
		"api": map[string]any{
			"version": h.APIVersion,
		},
		"runtime": map[string]any{
			"started_at":     h.StartedAt.UTC().Format(time.RFC3339),
			"uptime_seconds": int(time.Since(h.StartedAt).Seconds()),
		},
		"auth": map[string]any{
			"api_key":       h.HasAPIKeyAuth,
			"password_totp": h.HasPasswordAuth,
		},
		"notifications": notify,
	})
}

