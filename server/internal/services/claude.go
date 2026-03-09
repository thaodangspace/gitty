package services

import (
	"bytes"
	"fmt"
	"os/exec"
	"strings"
	"time"
)

type ClaudeService struct {
	timeout       time.Duration
	defaultPrompt string
}

func NewClaudeService(defaultPrompt string) *ClaudeService {
	return &ClaudeService{
		timeout:       60 * time.Second,
		defaultPrompt: defaultPrompt,
	}
}

func (s *ClaudeService) GenerateCommitMessage(diffs []string, customPrompt string) (string, error) {
	prompt := "generate commit message from changed files, only response commit message with jsonstringfy format (makesure Javascript can parse): {\"message\": \"<commit message>\", \"detail\": \"<detail>\"}"
	if customPrompt != "" {
		prompt = customPrompt
	}

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
