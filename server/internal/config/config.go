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

Provide only the commit message, no other text.`
}

func configFilePath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("determine home directory: %w", err)
	}

	return filepath.Join(home, ".config", "gitty.config.json"), nil
}
