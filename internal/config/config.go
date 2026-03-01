package config

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"

	"gtd/internal/models"
)

// LoadConfig loads configuration from disk or creates default
func LoadConfig() (*models.Config, error) {
	dataPath := getDataPath()
	configPath := filepath.Join(dataPath, "config.json")

	// Create data directory if it doesn't exist
	if err := os.MkdirAll(dataPath, 0755); err != nil {
		return nil, fmt.Errorf("failed to create data directory: %w", err)
	}

	// Load existing config or create default
	var cfg *models.Config
	created := false
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		cfg = defaultConfig()
		created = true
		if err := saveConfig(configPath, cfg); err != nil {
			return nil, err
		}
	} else {
		var err error
		cfg, err = loadConfig(configPath)
		if err != nil {
			return nil, err
		}
	}
	changed := applyDefaults(cfg, dataPath)

	cfg.DataPath = dataPath
	if created || changed {
		if err := saveConfig(configPath, cfg); err != nil {
			return nil, fmt.Errorf("failed to persist configuration defaults: %w", err)
		}
	}
	return cfg, nil
}

// loadConfig reads config from disk
func loadConfig(path string) (*models.Config, error) {
	data, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, err
	}

	raw := map[string]json.RawMessage{}
	if err := json.Unmarshal(data, &raw); err != nil {
		return nil, err
	}

	var cfg models.Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, err
	}
	if _, ok := raw["autoCommit"]; !ok {
		cfg.AutoCommit = true
	}

	return &cfg, nil
}

// SaveConfig persists config to disk
func SaveConfig(cfg *models.Config) error {
	return saveConfig(filepath.Join(cfg.DataPath, "config.json"), cfg)
}

// saveConfig writes config to disk
func saveConfig(path string, cfg *models.Config) error {
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}

	return ioutil.WriteFile(path, data, 0644)
}

// defaultConfig returns a new default configuration
func defaultConfig() *models.Config {
	return &models.Config{
		Contexts: []string{
			"work",
			"oscp",
			"sans-sec542",
			"home-office",
			"music",
			"woodworking",
			"research",
			"admin",
		},
		DefaultContext:      "work",
		AutoCommit:          true,
		AutoSync:            false,
		SyncIntervalMinutes: 30,
		GitRemote:           "origin",
		GitBranch:           "main",
		Mode:                "local",
	}
}

func applyDefaults(cfg *models.Config, dataPath string) bool {
	changed := false
	if len(cfg.Contexts) == 0 {
		cfg.Contexts = defaultConfig().Contexts
		changed = true
	}
	if cfg.DefaultContext == "" {
		cfg.DefaultContext = "work"
		changed = true
	}
	if cfg.SyncIntervalMinutes <= 0 {
		cfg.SyncIntervalMinutes = 30
		changed = true
	}
	if cfg.GitRemote == "" {
		cfg.GitRemote = "origin"
		changed = true
	}
	if cfg.GitBranch == "" {
		cfg.GitBranch = "main"
		changed = true
	}
	if cfg.Mode == "" {
		cfg.Mode = "local"
		changed = true
	}
	if cfg.APIKeyFile == "" {
		cfg.APIKeyFile = filepath.Join(dataPath, "credentials")
		changed = true
	}
	if cfg.LocalDBPath == "" {
		cfg.LocalDBPath = filepath.Join(dataPath, "gtd.db")
		changed = true
	}
	return changed
}

// getDataPath returns the path where GTD data is stored
func getDataPath() string {
	if dataPath := os.Getenv("GTD_DATA_PATH"); dataPath != "" {
		return dataPath
	}

	home, err := os.UserHomeDir()
	if err != nil {
		panic(err) // Should never happen in practice
	}

	return filepath.Join(home, ".gtd")
}
