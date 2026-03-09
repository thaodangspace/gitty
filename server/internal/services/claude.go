package services

import (
	"bytes"
	"fmt"
	"gitweb/server/internal/config"
	"os/exec"
	"strings"
	"time"
)

type ClaudeService struct {
	timeout time.Duration
	config  *config.Config
}

func NewClaudeService(cfg *config.Config) *ClaudeService {
	return &ClaudeService{
		timeout: 60 * time.Second,
		config:  cfg,
	}
}

func (s *ClaudeService) GenerateCommitMessage(diffs []string, customPrompt string) (string, error) {
	prompt := s.config.ClaudePromptValue()
	if customPrompt != "" {
		prompt = customPrompt
	}

	// Replace {{diffs}} placeholder with actual diffs
	diffsStr := strings.Join(diffs, "\n")
	prompt = strings.Replace(prompt, "{{diffs}}", diffsStr, 1)

	// Create claude command
	cmd := exec.Command("claude", prompt)

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	// Execute with timeout
	err := cmd.Start()
	if err != nil {
		return "", fmt.Errorf("failed to start claude command: %w", err)
	}

	done := make(chan error, 1)
	go func() {
		done <- cmd.Wait()
	}()

	select {
	case err := <-done:
		if err != nil {
			stderrStr := stderr.String()
			if stderrStr != "" {
				return "", fmt.Errorf("claude command failed: %s", stderrStr)
			}
			return "", fmt.Errorf("claude command failed: %w", err)
		}
	case <-time.After(s.timeout):
		cmd.Process.Kill()
		return "", fmt.Errorf("claude command timed out after %v", s.timeout)
	}

	message := strings.TrimSpace(stdout.String())

	// Clean up any markdown code blocks
	message = strings.TrimPrefix(message, "```")
	message = strings.TrimSuffix(message, "```")
	message = strings.TrimSpace(message)

	return message, nil
}
