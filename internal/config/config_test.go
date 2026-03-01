package config

import (
	"os"
	"testing"
)

func TestLoadConfigDefaults(t *testing.T) {
	dir := t.TempDir()
	if err := os.Setenv("GTD_DATA_PATH", dir); err != nil {
		t.Fatal(err)
	}
	defer os.Unsetenv("GTD_DATA_PATH")

	cfg, err := LoadConfig()
	if err != nil {
		t.Fatalf("LoadConfig failed: %v", err)
	}
	if cfg.DefaultContext == "" {
		t.Fatal("expected default context")
	}
	if cfg.SyncIntervalMinutes <= 0 {
		t.Fatal("expected sync interval default")
	}
}
