# AI-Powered Commit Message Generation Plan

## Context

This feature adds the ability to generate commit messages using the Claude Code CLI (`claude` command) when a user clicks a button on the commit dialog. The feature will:

1. Execute the `claude` CLI command as a subprocess from the Go backend
2. Pass staged file diffs to the CLI for analysis
3. Capture the generated commit message and return it to the frontend
4. Allow users to customize the prompt via configuration

## Implementation Overview

The implementation consists of:
- **Backend**: Go service that spawns `claude` subprocess, captures output
- **Configuration**: Server config with customizable prompt
- **Frontend**: Button in CommitDialog to trigger generation, loading states

## Files to Modify

### Backend Files
| File | Changes |
|------|---------|
| `server/internal/config/config.go` | Add `claudePrompt` field, validation, getter methods |
| `server/internal/api/handlers/repository.go` | Add `GenerateCommitMessage` handler, update constructor |
| `server/internal/api/routes.go` | Add new route `/api/repos/{id}/generate-commit-message` |
| `server/internal/services/claude.go` | **NEW** - Claude CLI service for subprocess execution |
| `server/internal/models/repository.go` | Add `GenerateCommitMessageResponse` type |

### Frontend Files
| File | Changes |
|------|---------|
| `www/src/types/api.ts` | Add `GenerateCommitMessageResponse` interface |
| `www/src/lib/api-client.ts` | Add `generateCommitMessage()` method |
| `www/src/store/queries/repository-queries.ts` | Add `useGenerateCommitMessage()` hook |
| `www/src/components/repository/CommitDialog.tsx` | Add sparkles button, integrate generation |

## Detailed Implementation

### Phase 1: Backend Configuration

#### 1.1 Update Config Struct

**File**: `server/internal/config/config.go`

Add `claudePrompt` field and related methods:

```go
type Config struct {
    MasterPassword *string `json:"masterPassword,omitempty"`
    ClaudePrompt   *string `json:"claudePrompt,omitempty"`
}

func (c *Config) Validate() error {
    // ... existing MasterPassword validation ...

    if c.ClaudePrompt != nil {
        *c.ClaudePrompt = strings.TrimSpace(*c.ClaudePrompt)
    }

    return nil
}

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
```

**Config file location**: `~/.config/gitty.config.json`

**Example config**:
```json
{
  "claudePrompt": "Write a commit message in format: type: description\n\nFiles changed:\n{{diffs}}"
}
```

### Phase 2: Claude CLI Service

#### 2.1 Create New Service File

**New File**: `server/internal/services/claude.go`

```go
package services

import (
    "bytes"
    "fmt"
    "os/exec"
    "strings"
    "time"
)

type ClaudeService struct {
    timeout   time.Duration
    defaultPrompt string
}

func NewClaudeService(defaultPrompt string) *ClaudeService {
    return &ClaudeService{
        timeout:   60 * time.Second,
        defaultPrompt: defaultPrompt,
    }
}

func (s *ClaudeService) GenerateCommitMessage(diffs []string, customPrompt string) (string, error) {
    if len(diffs) == 0 {
        return "", fmt.Errorf("no diffs provided")
    }

    // Build input for claude CLI
    diffContent := strings.Join(diffs, "\n\n---\n\n")
    prompt := s.defaultPrompt
    if customPrompt != "" {
        prompt = customPrompt
    }

    // Replace {{diffs}} placeholder with actual diffs
    fullPrompt := strings.ReplaceAll(prompt, "{{diffs}}", diffContent)

    // Create claude command
    cmd := exec.Command("claude", "ask", fullPrompt)

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
```

### Phase 3: Backend Handler

#### 3.1 Update RepositoryHandler

**File**: `server/internal/api/handlers/repository.go`

Add fields to struct and update constructor:

```go
type RepositoryHandler struct {
    gitService     *git.Service
    repositories   map[string]*models.Repository
    dataPath       string
    watcher        *git.RepositoryWatcher
    config         *config.Config
    claudeService  *services.ClaudeService
}

func NewRepositoryHandler(dataPath string, cfg *config.Config) *RepositoryHandler {
    // ... existing watcher initialization ...

    return &RepositoryHandler{
        gitService:     git.NewService(),
        repositories:   make(map[string]*models.Repository),
        dataPath:       dataPath,
        watcher:        watcher,
        config:         cfg,
        claudeService:  services.NewClaudeService(cfg.ClaudePromptValue()),
    }
}
```

Add new handler method:

```go
func (h *RepositoryHandler) GenerateCommitMessage(w http.ResponseWriter, r *http.Request) {
    repoID := chi.URLParam(r, "id")

    repo, exists := h.repositories[repoID]
    if !exists {
        http.Error(w, "Repository not found", http.StatusNotFound)
        return
    }

    // Get repository status to find staged files
    status, err := h.gitService.GetRepositoryStatus(repo.Path)
    if err != nil {
        http.Error(w, fmt.Sprintf("Failed to get repository status: %v", err), http.StatusInternalServerError)
        return
    }

    if len(status.Staged) == 0 {
        http.Error(w, "No staged files to generate commit message for", http.StatusBadRequest)
        return
    }

    // Collect diffs for all staged files
    var diffs []string
    for _, file := range status.Staged {
        diff, err := h.gitService.GetFileDiff(repo.Path, file.Path)
        if err != nil {
            http.Error(w, fmt.Sprintf("Failed to get diff for file %s: %v", file.Path, err), http.StatusInternalServerError)
            return
        }
        diffs = append(diffs, fmt.Sprintf("File: %s (Status: %s)\n%s", file.Path, file.Status, diff))
    }

    // Generate commit message using Claude CLI
    customPrompt := ""
    if h.config != nil && h.config.ClaudePrompt != nil {
        customPrompt = *h.config.ClaudePrompt
    }

    message, err := h.claudeService.GenerateCommitMessage(diffs, customPrompt)
    if err != nil {
        http.Error(w, fmt.Sprintf("Failed to generate commit message: %v", err), http.StatusInternalServerError)
        return
    }

    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(models.GenerateCommitMessageResponse{
        Message: message,
    })
}
```

### Phase 4: Backend Routes

#### 4.1 Add New Route

**File**: `server/internal/api/routes.go`

Add the new route in the `{id}` route group:

```go
r.Post("/generate-commit-message", repoHandler.GenerateCommitMessage)
```

Place it after the `CreateCommit` route.

### Phase 5: Backend Response Model

#### 5.1 Add Response Type

**File**: `server/internal/models/repository.go`

Add:

```go
type GenerateCommitMessageResponse struct {
    Message string `json:"message"`
}
```

### Phase 6: Frontend Types

#### 6.1 Update API Types

**File**: `www/src/types/api.ts`

Add:

```typescript
export interface GenerateCommitMessageResponse {
  message: string;
}
```

### Phase 7: Frontend API Client

#### 7.1 Add API Method

**File**: `www/src/lib/api-client.ts`

```typescript
async generateCommitMessage(id: string): Promise<GenerateCommitMessageResponse> {
    return this.request<GenerateCommitMessageResponse>(`/repos/${id}/generate-commit-message`, {
        method: 'POST',
    });
}
```

Import the type:

```typescript
import type { GenerateCommitMessageResponse } from '../types/api';
```

### Phase 8: Frontend React Query Hook

#### 8.1 Add Mutation Hook

**File**: `www/src/store/queries/repository-queries.ts`

```typescript
export const useGenerateCommitMessage = () => {
    return useMutation({
        mutationFn: ({ repositoryId }: { repositoryId: string }) =>
            apiClient.generateCommitMessage(repositoryId),
    });
};
```

### Phase 9: Frontend UI Changes

#### 9.1 Update CommitDialog

**File**: `www/src/components/repository/CommitDialog.tsx`

**Add imports**:
```typescript
import { Sparkles } from 'lucide-react';
import { useGenerateCommitMessage } from '@/store/queries';
```

**Add mutation hook** (after line 36):
```typescript
const generateCommitMessageMutation = useGenerateCommitMessage();
```

**Add handler function** (after line 96):
```typescript
const handleGenerateMessage = async () => {
    if (!currentRepository || stagedFilesCount === 0) {
        return;
    }

    try {
        const result = await generateCommitMessageMutation.mutateAsync({
            repositoryId: currentRepository.id,
        });
        setCommitMessage(result.message);
    } catch (error) {
        console.error('Failed to generate commit message:', error);
    }
};
```

**Update commit message input section** (replace lines 122-134):
```typescript
<div className="grid gap-2">
    <label htmlFor="commitMessage" className="text-sm font-medium">
        Commit message *
    </label>
    <div className="flex gap-2">
        <Input
            id="commitMessage"
            value={commitMessage}
            onChange={(e) => setCommitMessage(e.target.value)}
            placeholder="Add a meaningful commit message..."
            disabled={createCommitMutation.isPending}
            autoFocus
            className="flex-1"
        />
        <Button
            type="button"
            variant="outline"
            size="icon"
            onClick={handleGenerateMessage}
            disabled={
                stagedFilesCount === 0 ||
                generateCommitMessageMutation.isPending ||
                createCommitMutation.isPending
            }
            title="Generate commit message with AI"
        >
            {generateCommitMessageMutation.isPending ? (
                <Loader2 className="h-4 w-4 animate-spin" />
            ) : (
                <Sparkles className="h-4 w-4" />
            )}
        </Button>
    </div>
</div>
```

**Add error display** (after line 184):
```typescript
{generateCommitMessageMutation.isError && (
    <p className="text-sm text-amber-600">
        Failed to generate commit message: {generateCommitMessageMutation.error.message}
    </p>
)}
```

## Verification

### Manual Testing Steps

1. **Setup**:
   - Ensure `claude` CLI is installed and authenticated
   - Optionally create `~/.config/gitty.config.json` with custom prompt

2. **Test Empty Staged Files**:
   - Open commit dialog with no staged files
   - Verify generate button is disabled
   - Expected: Button shows as disabled

3. **Test Successful Generation**:
   - Stage some files in a repository
   - Open commit dialog
   - Click the sparkles button
   - Verify loading spinner appears
   - Verify commit message is populated after generation
   - Expected: Button shows loader, then message appears in input

4. **Test Error Handling**:
   - Temporarily rename `claude` binary to simulate missing command
   - Try to generate commit message
   - Verify error message is displayed in amber color
   - Expected: Amber error message shown

5. **Test Custom Prompt**:
   - Set custom prompt in config file
   - Restart server
   - Generate commit message
   - Verify format matches custom prompt
   - Expected: Message follows custom format

### Backend Testing

```bash
# Test the endpoint directly
curl -X POST http://localhost:8083/api/repos/{repo-id}/generate-commit-message

# Expected response:
{
  "message": "feat(api): add commit message generation"
}
```

## Reusable Components/Functions

The following existing functions are reused:
- `gitService.GetRepositoryStatus()` - Get staged files
- `gitService.GetFileDiff()` - Get individual file diffs
- `apiClient.request()` - Standard API call pattern
- `useMutation()` - React Query mutation pattern
- `Loader2` icon - Existing loading indicator
- `Button` component - shadcn/ui button with variants

## Error Handling

| Scenario | Status Code | Frontend Behavior |
|----------|-------------|-------------------|
| Repository not found | 404 | Error message shown |
| No staged files | 400 | Generate button disabled |
| Claude command not found | 500 | Amber error message |
| Claude command timeout | 500 | Timeout error message |
| Claude CLI authentication error | 500 | Authentication error message |

## Notes

- The `claude` CLI must be installed on the server and accessible in PATH
- User must be authenticated with Claude Code CLI
- Default prompt uses conventional commit format
- The prompt supports `{{diffs}}` placeholder for the actual diffs
- 60-second timeout prevents hanging subprocesses
