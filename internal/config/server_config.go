package config

import (
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
)

// ServerConfig stores file-based server settings for the upcoming web/API runtime.
type ServerConfig struct {
	ListenAddr   string `json:"listen_addr"`
	JWTSecret    string `json:"jwt_secret"`
	TOTPIssuer   string `json:"totp_issuer"`
	DBPath       string `json:"db_path"`
	APITokenName string `json:"api_token_name"`
	CookieSecure bool   `json:"cookie_secure"`
	CSRFCookie   string `json:"csrf_cookie_name"`
	NotificationsEnabled      bool   `json:"notifications_enabled"`
	NotificationCheckSeconds  int    `json:"notification_check_seconds"`
	NotificationDryRun        bool   `json:"notification_dry_run"`
	NotificationEmailTo       string `json:"notification_email_to"`
	NotificationEmailFrom     string `json:"notification_email_from"`
	NotificationSMTPHost      string `json:"notification_smtp_host"`
	NotificationSMTPPort      int    `json:"notification_smtp_port"`
	NotificationSMTPUsername  string `json:"notification_smtp_username"`
	NotificationSMTPPassword  string `json:"notification_smtp_password"`
}

// LoadServerConfig loads or creates ~/.gtd/server-config.json.
func LoadServerConfig() (*ServerConfig, error) {
	dataPath := getDataPath()
	configPath := filepath.Join(dataPath, "server-config.json")

	if err := os.MkdirAll(dataPath, 0o755); err != nil {
		return nil, fmt.Errorf("failed to create data directory: %w", err)
	}

	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		cfg, err := defaultServerConfig(dataPath)
		if err != nil {
			return nil, err
		}
		if err := SaveServerConfig(cfg); err != nil {
			return nil, err
		}
		return cfg, nil
	}

	data, err := ioutil.ReadFile(configPath)
	if err != nil {
		return nil, err
	}
	var cfg ServerConfig
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, err
	}
	defaults, err := defaultServerConfig(dataPath)
	if err != nil {
		return nil, err
	}
	changed := false
	if cfg.ListenAddr == "" {
		cfg.ListenAddr = defaults.ListenAddr
		changed = true
	}
	if cfg.DBPath == "" {
		cfg.DBPath = defaults.DBPath
		changed = true
	}
	if cfg.JWTSecret == "" {
		cfg.JWTSecret = defaults.JWTSecret
		changed = true
	}
	if cfg.TOTPIssuer == "" {
		cfg.TOTPIssuer = defaults.TOTPIssuer
		changed = true
	}
	if cfg.APITokenName == "" {
		cfg.APITokenName = defaults.APITokenName
		changed = true
	}
	if cfg.CSRFCookie == "" {
		cfg.CSRFCookie = defaults.CSRFCookie
		changed = true
	}
	if cfg.NotificationCheckSeconds <= 0 {
		cfg.NotificationCheckSeconds = defaults.NotificationCheckSeconds
		changed = true
	}
	if cfg.NotificationSMTPPort <= 0 {
		cfg.NotificationSMTPPort = defaults.NotificationSMTPPort
		changed = true
	}
	if cfg.NotificationEmailFrom == "" {
		cfg.NotificationEmailFrom = defaults.NotificationEmailFrom
		changed = true
	}
	if changed {
		if err := SaveServerConfig(&cfg); err != nil {
			return nil, err
		}
	}
	return &cfg, nil
}

// SaveServerConfig persists server config to ~/.gtd/server-config.json.
func SaveServerConfig(cfg *ServerConfig) error {
	path := filepath.Join(getDataPath(), "server-config.json")
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}
	return ioutil.WriteFile(path, data, 0o644)
}

func defaultServerConfig(dataPath string) (*ServerConfig, error) {
	secret, err := generateSecret()
	if err != nil {
		return nil, err
	}
	return &ServerConfig{
		ListenAddr:   "127.0.0.1:8080",
		JWTSecret:    secret,
		TOTPIssuer:   "GTD@eatbrainz.com",
		DBPath:       filepath.Join(dataPath, "gtd.db"),
		APITokenName: "gtd_api_key",
		CookieSecure: false,
		CSRFCookie:   "gtd_csrf",
		NotificationsEnabled:     false,
		NotificationCheckSeconds: 300,
		NotificationDryRun:       true,
		NotificationEmailTo:      "",
		NotificationEmailFrom:    "gtd@localhost",
		NotificationSMTPHost:     "",
		NotificationSMTPPort:     587,
		NotificationSMTPUsername: "",
		NotificationSMTPPassword: "",
	}, nil
}

func generateSecret() (string, error) {
	buf := make([]byte, 32)
	if _, err := rand.Read(buf); err != nil {
		return "", fmt.Errorf("generate secret: %w", err)
	}
	return base64.StdEncoding.EncodeToString(buf), nil
}

