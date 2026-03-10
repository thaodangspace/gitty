# Plan: Fix diff endpoint for files with dots in names

## Context
The `/api/repos/{id}/diff/{path}` endpoint doesn't work for files with dots in their names (like `.claude.json`). The issue is that file paths are not being properly URL-encoded when sent to the server, and not being decoded when received.

## Problem
- Frontend (`www/src/lib/api-client.ts`) constructs diff URLs without encoding the file path
- Backend (`server/internal/api/handlers/repository.go`) extracts the wildcard path without URL-decoding it
- Files like `.claude.json` fail because the path is not properly handled

## Solution

### 1. Frontend fix - `www/src/lib/api-client.ts`
Update `getFileDiff()` method to encode the file path:
```typescript
async getFileDiff(id: string, filePath: string): Promise<string> {
    const encodedPath = encodeURIComponent(filePath);
    const response = await fetch(`${API_BASE_URL}/repos/${id}/diff/${encodedPath}`);
    // ... rest of the method
}
```

Also check and fix similar methods that use file paths in URLs:
- `getFileContent()` (line 177-186)
- `saveFileContent()` (line 188-196)
- `stageFile()` (line 154-158)
- `unstageFile()` (line 166-170)

### 2. Backend fix - `server/internal/api/handlers/repository.go`
Update handlers to decode the wildcard path parameter:

For `GetFileDiff()` (line 651-674):
```go
filePath := chi.URLParam(r, "*")
decodedPath, err := url.PathUnescape(filePath)
if err != nil {
    decodedPath = filePath // fallback to original if decoding fails
}
```

Apply same fix to:
- `GetFileContent()` (line 377-395)
- `SaveFileContent()` (line 397-421)
- `StageFile()` (line 539-557)
- `UnstageFile()` (line 578-596)

## Files to modify
1. `www/src/lib/api-client.ts` - Add URL encoding to file path methods
2. `server/internal/api/handlers/repository.go` - Add URL decoding to handlers

## Verification
1. Start the server and frontend
2. Open a repository that contains `.claude.json` or similar dotfiles
3. Try to view the diff for the file
4. Verify the diff is displayed correctly
5. Also test other file operations (view content, stage, unstage) with dotfiles
