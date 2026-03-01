package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
)

// NotificationRunner triggers reminder scans.
type NotificationRunner interface {
	RunOnce(ctx context.Context) (NotificationRunStats, error)
}

// NotificationRunStats summarizes reminder execution.
type NotificationRunStats struct {
	Scanned   int `json:"scanned"`
	Attempted int `json:"attempted"`
	Sent      int `json:"sent"`
	DryRun    int `json:"dry_run"`
	Failed    int `json:"failed"`
	Skipped   int `json:"skipped"`
}

// NotificationConfig represents runtime notification settings.
type NotificationConfig struct {
	Enabled      bool   `json:"enabled"`
	CheckSeconds int    `json:"check_seconds"`
	DryRun       bool   `json:"dry_run"`
	EmailTo      string `json:"email_to"`
	EmailFrom    string `json:"email_from"`
	SMTPHost     string `json:"smtp_host"`
	SMTPPort     int    `json:"smtp_port"`
	SMTPUsername string `json:"smtp_username"`
	HasSMTPPassword bool `json:"has_smtp_password"`
	RestartRequired bool `json:"restart_required"`
}

// NotificationConfigStore gets and persists config updates.
type NotificationConfigStore interface {
	GetNotificationConfig() NotificationConfig
	UpdateNotificationConfig(ctx context.Context, next NotificationConfigUpdate) (NotificationConfig, error)
}

// NotificationConfigUpdate is patch-style input for config mutation.
type NotificationConfigUpdate struct {
	Enabled      *bool   `json:"enabled"`
	CheckSeconds *int    `json:"check_seconds"`
	DryRun       *bool   `json:"dry_run"`
	EmailTo      *string `json:"email_to"`
	EmailFrom    *string `json:"email_from"`
	SMTPHost     *string `json:"smtp_host"`
	SMTPPort     *int    `json:"smtp_port"`
	SMTPUsername *string `json:"smtp_username"`
	SMTPPassword *string `json:"smtp_password"`
}

// NotificationsHandler exposes notification controls.
type NotificationsHandler struct {
	Runner NotificationRunner
	Store  NotificationConfigStore
}

func (h *NotificationsHandler) RunNow(w http.ResponseWriter, r *http.Request) {
	if h.Runner == nil {
		writeError(w, http.StatusNotImplemented, "notification runner not configured")
		return
	}
	stats, err := h.Runner.RunOnce(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"ok":    true,
		"stats": stats,
	})
}

func (h *NotificationsHandler) GetConfig(w http.ResponseWriter, _ *http.Request) {
	if h.Store == nil {
		writeError(w, http.StatusNotImplemented, "notification config not configured")
		return
	}
	writeJSON(w, http.StatusOK, h.Store.GetNotificationConfig())
}

func (h *NotificationsHandler) UpdateConfig(w http.ResponseWriter, r *http.Request) {
	if h.Store == nil {
		writeError(w, http.StatusNotImplemented, "notification config not configured")
		return
	}
	var in NotificationConfigUpdate
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		writeError(w, http.StatusBadRequest, "invalid json body")
		return
	}
	cfg, err := h.Store.UpdateNotificationConfig(r.Context(), sanitizeUpdate(in))
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, cfg)
}

func sanitizeUpdate(in NotificationConfigUpdate) NotificationConfigUpdate {
	if in.EmailTo != nil {
		v := strings.TrimSpace(*in.EmailTo)
		in.EmailTo = &v
	}
	if in.EmailFrom != nil {
		v := strings.TrimSpace(*in.EmailFrom)
		in.EmailFrom = &v
	}
	if in.SMTPHost != nil {
		v := strings.TrimSpace(*in.SMTPHost)
		in.SMTPHost = &v
	}
	if in.SMTPUsername != nil {
		v := strings.TrimSpace(*in.SMTPUsername)
		in.SMTPUsername = &v
	}
	return in
}

func validateNotificationConfig(cfg NotificationConfig) error {
	if cfg.CheckSeconds <= 0 {
		return fmt.Errorf("check_seconds must be > 0")
	}
	if cfg.SMTPPort < 0 {
		return fmt.Errorf("smtp_port must be >= 0")
	}
	if !cfg.DryRun {
		if cfg.EmailTo == "" || cfg.EmailFrom == "" || cfg.SMTPHost == "" || cfg.SMTPPort == 0 {
			return fmt.Errorf("smtp configuration is required when dry_run is false")
		}
	}
	return nil
}

