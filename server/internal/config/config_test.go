package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadConfigDefaultsResourceGovernor(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("HOME", tmp)

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() returned error: %v", err)
	}

	if cfg.ResourceGovernor == nil {
		t.Fatalf("expected resource governor config to be initialized")
	}

	if !cfg.ResourceGovernor.Enabled {
		t.Fatalf("expected resource governor to be enabled by default")
	}

	if got, want := cfg.ResourceGovernor.MemoryLimitBytes, int64(1<<30); got != want {
		t.Fatalf("expected memory limit %d, got %d", want, got)
	}

	if got, want := cfg.ResourceGovernor.GOMAXPROCS, 2; got != want {
		t.Fatalf("expected gomaxprocs %d, got %d", want, got)
	}

	if got, want := cfg.ResourceGovernor.MaxExpensiveInflight, 10; got != want {
		t.Fatalf("expected max expensive inflight %d, got %d", want, got)
	}

	if got, want := cfg.ResourceGovernor.SampleIntervalMs, 500; got != want {
		t.Fatalf("expected sample interval %d, got %d", want, got)
	}

	if got, want := cfg.ResourceGovernor.DegradeHighWatermark, 0.85; got != want {
		t.Fatalf("expected high watermark %v, got %v", want, got)
	}

	if got, want := cfg.ResourceGovernor.DegradeLowWatermark, 0.70; got != want {
		t.Fatalf("expected low watermark %v, got %v", want, got)
	}

	if got, want := cfg.ResourceGovernor.RetryAfterSeconds, 3; got != want {
		t.Fatalf("expected retry-after seconds %d, got %d", want, got)
	}
}

func TestLoadConfigWithResourceGovernorOverrides(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("HOME", tmp)

	configDir := filepath.Join(tmp, ".config")
	if err := os.MkdirAll(configDir, 0o755); err != nil {
		t.Fatalf("failed to create config directory: %v", err)
	}

	configPath := filepath.Join(configDir, "gitty.config.json")
	if err := os.WriteFile(configPath, []byte(`{
		"resourceGovernor": {
			"enabled": true,
			"memoryLimitBytes": 2147483648,
			"gomaxprocs": 4,
			"maxExpensiveInflight": 6,
			"sampleIntervalMs": 1000,
			"degradeHighWatermark": 0.9,
			"degradeLowWatermark": 0.75,
			"retryAfterSeconds": 10
		}
	}`), 0o600); err != nil {
		t.Fatalf("failed to write config file: %v", err)
	}

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() returned error: %v", err)
	}

	if cfg.ResourceGovernor == nil {
		t.Fatalf("expected resource governor config to be initialized")
	}

	if !cfg.ResourceGovernor.Enabled {
		t.Fatalf("expected resource governor to be enabled")
	}

	if got, want := cfg.ResourceGovernor.MemoryLimitBytes, int64(2147483648); got != want {
		t.Fatalf("expected memory limit %d, got %d", want, got)
	}

	if got, want := cfg.ResourceGovernor.GOMAXPROCS, 4; got != want {
		t.Fatalf("expected gomaxprocs %d, got %d", want, got)
	}

	if got, want := cfg.ResourceGovernor.MaxExpensiveInflight, 6; got != want {
		t.Fatalf("expected max expensive inflight %d, got %d", want, got)
	}

	if got, want := cfg.ResourceGovernor.SampleIntervalMs, 1000; got != want {
		t.Fatalf("expected sample interval %d, got %d", want, got)
	}

	if got, want := cfg.ResourceGovernor.DegradeHighWatermark, 0.9; got != want {
		t.Fatalf("expected high watermark %v, got %v", want, got)
	}

	if got, want := cfg.ResourceGovernor.DegradeLowWatermark, 0.75; got != want {
		t.Fatalf("expected low watermark %v, got %v", want, got)
	}

	if got, want := cfg.ResourceGovernor.RetryAfterSeconds, 10; got != want {
		t.Fatalf("expected retry-after seconds %d, got %d", want, got)
	}
}

func TestLoadConfigInvalidResourceGovernorWatermarks(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("HOME", tmp)

	configDir := filepath.Join(tmp, ".config")
	if err := os.MkdirAll(configDir, 0o755); err != nil {
		t.Fatalf("failed to create config directory: %v", err)
	}

	configPath := filepath.Join(configDir, "gitty.config.json")
	if err := os.WriteFile(configPath, []byte(`{
		"resourceGovernor": {
			"degradeHighWatermark": 0.8,
			"degradeLowWatermark": 0.9
		}
	}`), 0o600); err != nil {
		t.Fatalf("failed to write config file: %v", err)
	}

	if _, err := Load(); err == nil {
		t.Fatalf("expected error when resource governor watermarks are invalid")
	}
}

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

	if cfg.ResourceGovernor == nil {
		t.Fatalf("expected resource governor config to be initialized")
	}

	if !cfg.ResourceGovernor.Enabled {
		t.Fatalf("expected resource governor to be enabled by default")
	}
}

func TestValidateInitializesDefaultResourceGovernor(t *testing.T) {
	cfg := &Config{}

	if err := cfg.Validate(); err != nil {
		t.Fatalf("Validate() returned error: %v", err)
	}

	if cfg.ResourceGovernor == nil {
		t.Fatalf("expected resource governor config to be initialized")
	}

	if !cfg.ResourceGovernor.Enabled {
		t.Fatalf("expected resource governor to be enabled by default")
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

func TestRequireMasterPassword(t *testing.T) {
	t.Run("returns error when master password is not configured", func(t *testing.T) {
		cfg := &Config{}
		if err := cfg.Validate(); err != nil {
			t.Fatalf("Validate() returned error: %v", err)
		}

		password, err := cfg.RequireMasterPassword()
		if err == nil {
			t.Fatalf("expected error when master password is not configured")
		}
		if password != "" {
			t.Fatalf("expected empty password on error, got %q", password)
		}
	})

	t.Run("returns password when configured", func(t *testing.T) {
		tmp := t.TempDir()
		t.Setenv("HOME", tmp)

		configDir := filepath.Join(tmp, ".config")
		if err := os.MkdirAll(configDir, 0o755); err != nil {
			t.Fatalf("failed to create config directory: %v", err)
		}

		configPath := filepath.Join(configDir, "gitty.config.json")
		if err := os.WriteFile(configPath, []byte(`{"masterPassword":"secret"}`), 0o600); err != nil {
			t.Fatalf("failed to write config file: %v", err)
		}

		cfg, err := Load()
		if err != nil {
			t.Fatalf("Load() returned error: %v", err)
		}

		password, err := cfg.RequireMasterPassword()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if password != "secret" {
			t.Fatalf("expected password 'secret', got %q", password)
		}
	})
}
