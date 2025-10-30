package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadConfigWithoutFile(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("HOME", tmp)

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() returned error: %v", err)
	}

	if cfg.HasMasterPassword() {
		t.Fatalf("expected no master password when file is missing")
	}
}

func TestLoadConfigWithMasterPassword(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("HOME", tmp)

	configDir := filepath.Join(tmp, ".config")
	if err := os.MkdirAll(configDir, 0o755); err != nil {
		t.Fatalf("failed to create config directory: %v", err)
	}

	configPath := filepath.Join(configDir, "gitty.config.json")
	if err := os.WriteFile(configPath, []byte(`{"masterPassword":"  secret  "}`), 0o600); err != nil {
		t.Fatalf("failed to write config file: %v", err)
	}

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() returned error: %v", err)
	}

	password, ok := cfg.MasterPasswordValue()
	if !ok {
		t.Fatalf("expected master password to be enabled")
	}

	if password != "secret" {
		t.Fatalf("expected password to be trimmed to 'secret', got %q", password)
	}
}

func TestLoadConfigWithEmptyMasterPassword(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("HOME", tmp)

	configDir := filepath.Join(tmp, ".config")
	if err := os.MkdirAll(configDir, 0o755); err != nil {
		t.Fatalf("failed to create config directory: %v", err)
	}

	configPath := filepath.Join(configDir, "gitty.config.json")
	if err := os.WriteFile(configPath, []byte(`{"masterPassword":""}`), 0o600); err != nil {
		t.Fatalf("failed to write config file: %v", err)
	}

	if _, err := Load(); err == nil {
		t.Fatalf("expected error when master password is empty")
	}
}
