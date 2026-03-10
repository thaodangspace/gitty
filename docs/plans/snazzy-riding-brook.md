# Prefill Git User on Commit Popup

## Context

Currently, when users open the commit dialog, the author name and email fields are empty. Users must manually enter their git credentials each time, or leave them blank (which causes git to use its default behavior). This plan adds functionality to:
1. Fetch the current git config `user.name` and `user.email` from the backend
2. Pre-populate these values in the commit dialog
3. Provide a checkbox to allow users to override the pre-filled values

This improves UX by reducing repetitive input while still allowing customization.

## Implementation Plan

### Backend Changes

#### 1. Add GitConfig model (`server/internal/models/repository.go`)
Add a new struct to hold git config user information:
```go
type GitConfig struct {
    Name  string `json:"name"`
    Email string `json:"email"`
}
```

#### 2. Add service method to get git config (`server/internal/git/service.go`)
Add a new method `GetGitConfig(repoPath string) (*GitConfig, error)` that:
- Opens the repository using `git.PlainOpen`
- Reads the config using go-git's `repo.Config()`
- Returns `user.name` and `user.email` (may be empty if not set)

Note: go-git's `Config()` method returns a `*config.Config` with `User.Name` and `User.Email` fields.

#### 3. Add API handler (`server/internal/api/handlers/repository.go`)
Add a new handler method `GetGitConfig(w http.ResponseWriter, r *http.Request)`:
- Extracts repo ID from URL
- Gets the repository path
- Calls `gitService.GetGitConfig`
- Returns JSON response with name and email

#### 4. Add API route (`server/internal/api/routes.go`)
Add new route: `r.Get("/config/git", repoHandler.GetGitConfig)` inside the `/{id}` route block.

### Frontend Changes

#### 1. Add types (`www/src/types/api.ts`)
Add new interface:
```typescript
export interface GitConfig {
  name: string;
  email: string;
}
```

#### 2. Add API client method (`www/src/lib/api-client.ts`)
Add method `getGitConfig(id: string): Promise<GitConfig>` that calls `/repos/{id}/config/git`.

#### 3. Add React Query hook (`www/src/store/queries/repository-queries.ts`)
Create `useGitConfig` hook using `useQuery` that:
- Fetches git config when repository is selected
- Has appropriate stale time (can be fairly long since git config rarely changes)

#### 4. Update CommitDialog component (`www/src/components/repository/CommitDialog.tsx`)
Changes needed:
- Import `useGitConfig` hook
- Add state `showCustomAuthor` (boolean) to toggle between prefilled and custom values
- On dialog open, fetch git config and populate `authorName` and `authorEmail` state
- Render a checkbox "Use different author" that when checked, shows the input fields
- When checkbox is unchecked, clear the author fields (or use the git config values)
- Only send author in commit request if custom author is explicitly provided

UI layout suggestion:
```
[ ] Use different author
  (When checked, show name/email inputs below)
```

### Critical Files to Modify

1. `server/internal/models/repository.go` - Add GitConfig struct
2. `server/internal/git/service.go` - Add GetGitConfig method
3. `server/internal/api/handlers/repository.go` - Add GetGitConfig handler
4. `server/internal/api/routes.go` - Add /config/git route
5. `www/src/types/api.ts` - Add GitConfig type
6. `www/src/lib/api-client.ts` - Add getGitConfig method
7. `www/src/store/queries/repository-queries.ts` - Add useGitConfig hook
8. `www/src/components/repository/CommitDialog.tsx` - Update UI with checkbox logic

### Verification Steps

1. **Backend testing**:
   - Start the backend server
   - Use curl/Postman to test `GET /api/repos/{id}/config/git`
   - Verify it returns the correct git user.name and user.email from the repository's config

2. **Frontend testing**:
   - Open the commit dialog for a repository
   - Verify the author fields are pre-filled with git config values
   - Verify the checkbox "Use different author" is present
   - Check the box and verify input fields appear
   - Enter custom values and create a commit
   - Verify the commit uses the custom author information

3. **Edge cases**:
   - Test with a repo that has no git config set (should show empty fields or handle gracefully)
   - Test with global git config vs local repo config
