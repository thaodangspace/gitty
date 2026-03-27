package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// Config represents the server configuration that can be customized via
// ~/.config/gitty.config.json.
type Config struct {
	// MasterPassword, when provided, enables password protection for the
	// server. A nil value indicates password protection is disabled.
	MasterPassword *string `json:"masterPassword,omitempty"`
	// ClaudePrompt, when provided, customizes the prompt used by the Claude CLI
	// to generate commit messages. If nil, a default prompt is used.
	ClaudePrompt *string `json:"claudePrompt,omitempty"`
	// ResourceGovernor controls memory and concurrency-related guardrails.
	ResourceGovernor *ResourceGovernorConfig `json:"resourceGovernor,omitempty"`
}

// ResourceGovernorConfig contains the resource governor limits and thresholds.
type ResourceGovernorConfig struct {
	Enabled              bool    `json:"enabled"`
	MemoryLimitBytes     int64   `json:"memoryLimitBytes"`
	GOMAXPROCS           int     `json:"gomaxprocs"`
	MaxExpensiveInflight int     `json:"maxExpensiveInflight"`
	SampleIntervalMs     int     `json:"sampleIntervalMs"`
	DegradeHighWatermark float64 `json:"degradeHighWatermark"`
	DegradeLowWatermark  float64 `json:"degradeLowWatermark"`
	RetryAfterSeconds    int     `json:"retryAfterSeconds"`
}

// Load reads the configuration from ~/.config/gitty.config.json. If the file
// does not exist a default configuration is returned.
func Load() (*Config, error) {
	configPath, err := configFilePath()
	if err != nil {
		return nil, err
	}

	cfg := &Config{}

	data, err := os.ReadFile(configPath)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			if err := cfg.Validate(); err != nil {
				return nil, err
			}
			return cfg, nil
		}
		return nil, fmt.Errorf("read config file: %w", err)
	}

	if err := json.Unmarshal(data, cfg); err != nil {
		return nil, fmt.Errorf("decode config file: %w", err)
	}

	if err := cfg.Validate(); err != nil {
		return nil, err
	}

	return cfg, nil
}

// Validate verifies that any optional fields, when provided, contain a usable
// value.
func (c *Config) Validate() error {
	if c.MasterPassword != nil {
		trimmed := strings.TrimSpace(*c.MasterPassword)
		if trimmed == "" {
			return errors.New("masterPassword cannot be empty when provided")
		}
		*c.MasterPassword = trimmed
	}

	if c.ClaudePrompt != nil {
		*c.ClaudePrompt = strings.TrimSpace(*c.ClaudePrompt)
	}

	if c.ResourceGovernor == nil {
		c.ResourceGovernor = defaultResourceGovernorConfig()
	} else {
		applyResourceGovernorDefaults(c.ResourceGovernor)
	}

	if err := c.ResourceGovernor.validate(); err != nil {
		return err
	}

	return nil
}

// HasMasterPassword reports whether password protection is enabled. Other
// modules can use this to determine if authentication should be enforced.
func (c Config) HasMasterPassword() bool {
	return c.MasterPassword != nil && strings.TrimSpace(*c.MasterPassword) != ""
}

// MasterPasswordValue returns the configured master password and a boolean
// indicating whether password protection is enabled.
func (c Config) MasterPasswordValue() (string, bool) {
	if !c.HasMasterPassword() {
		return "", false
	}
	return *c.MasterPassword, true
}

// ClaudePromptValue returns the configured Claude prompt, or a default prompt
// if none is configured.
func (c Config) ClaudePromptValue() string {
	if c.ClaudePrompt != nil && strings.TrimSpace(*c.ClaudePrompt) != "" {
		return *c.ClaudePrompt
	}
	// Default prompt
	return `You are a helpful assistant that writes Git commit messages.

Given the following file diffs from staged changes, write a meaningful commit message following the conventional commit format.

Format your response as:
<type>(<scope>): <subject>

<body>

<footer>

Where type is one of: feat, fix, docs, style, refactor, test, chore, perf, ci, build, revert

Keep the subject under 72 characters and imperative mood (e.g., "add feature" not "added feature").

Here are the diffs:

{{diffs}}

	Provide only the commit message, no other text. response with jsonstringfy format {message: <message>, detail: <detail>}`
}

func defaultResourceGovernorConfig() *ResourceGovernorConfig {
	return &ResourceGovernorConfig{
		Enabled:              false,
		MemoryLimitBytes:     1 << 30,
		GOMAXPROCS:           2,
		MaxExpensiveInflight: 2,
		SampleIntervalMs:     500,
		DegradeHighWatermark: 0.85,
		DegradeLowWatermark:  0.70,
		RetryAfterSeconds:    3,
	}
}

func applyResourceGovernorDefaults(cfg *ResourceGovernorConfig) {
	defaults := defaultResourceGovernorConfig()

	if cfg.MemoryLimitBytes == 0 {
		cfg.MemoryLimitBytes = defaults.MemoryLimitBytes
	}
	if cfg.GOMAXPROCS == 0 {
		cfg.GOMAXPROCS = defaults.GOMAXPROCS
	}
	if cfg.MaxExpensiveInflight == 0 {
		cfg.MaxExpensiveInflight = defaults.MaxExpensiveInflight
	}
	if cfg.SampleIntervalMs == 0 {
		cfg.SampleIntervalMs = defaults.SampleIntervalMs
	}
	if cfg.DegradeHighWatermark == 0 {
		cfg.DegradeHighWatermark = defaults.DegradeHighWatermark
	}
	if cfg.DegradeLowWatermark == 0 {
		cfg.DegradeLowWatermark = defaults.DegradeLowWatermark
	}
	if cfg.RetryAfterSeconds == 0 {
		cfg.RetryAfterSeconds = defaults.RetryAfterSeconds
	}
}

func (c ResourceGovernorConfig) validate() error {
	if c.MemoryLimitBytes <= 0 {
		return errors.New("resourceGovernor.memoryLimitBytes must be positive")
	}
	if c.GOMAXPROCS <= 0 {
		return errors.New("resourceGovernor.gomaxprocs must be positive")
	}
	if c.MaxExpensiveInflight <= 0 {
		return errors.New("resourceGovernor.maxExpensiveInflight must be positive")
	}
	if c.SampleIntervalMs <= 0 {
		return errors.New("resourceGovernor.sampleIntervalMs must be positive")
	}
	if c.DegradeHighWatermark <= 0 || c.DegradeHighWatermark > 1 {
		return errors.New("resourceGovernor.degradeHighWatermark must be in (0, 1]")
	}
	if c.DegradeLowWatermark <= 0 || c.DegradeLowWatermark > 1 {
		return errors.New("resourceGovernor.degradeLowWatermark must be in (0, 1]")
	}
	if c.DegradeLowWatermark >= c.DegradeHighWatermark {
		return errors.New("resourceGovernor.degradeLowWatermark must be less than resourceGovernor.degradeHighWatermark")
	}
	if c.RetryAfterSeconds <= 0 {
		return errors.New("resourceGovernor.retryAfterSeconds must be positive")
	}

	return nil
}

func configFilePath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("determine home directory: %w", err)
	}

	return filepath.Join(home, ".config", "gitty.config.json"), nil
}
