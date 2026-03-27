# Repo Settings Backend Design

**Date:** 2026-03-27
**Status:** Approved

## Goal
Implement backend support for repo settings endpoints used by the mobile app, combining Git-backed values with server-side app preferences.

## API

Add endpoints:
- `GET /api/repos/{id}/settings`
- `PUT /api/repos/{id}/settings/identity`
- `PUT /api/repos/{id}/settings/sync`
- `PUT /api/repos/{id}/settings/commit`

Keep existing endpoint unchanged for compatibility:
- `GET /api/repos/{id}/config/git`

## Response Shape

```json
{
  "identity": {
    "name": "string",
    "email": "string"
  },
  "sync": {
    "autoFetch": true,
    "fetchIntervalMinutes": 15,
    "pullStrategy": "merge"
  },
  "remotes": [
    { "name": "origin", "url": "https://example.com/repo.git" }
  ],
  "commit": {
    "defaultBranch": "main",
    "signingEnabled": false,
    "lineEndings": "lf"
  }
}
```

## Persistence Model

Use dedicated per-repo settings files.

- Directory: `<dataPath>/settings/`
- File: `<dataPath>/settings/<repoID>.json`

Store only app-level preferences in file:
- `sync`
- `commit`

Do not store:
- `identity` (written/read from `.git/config`)
- `remotes` (read from git remotes)

## Defaults

If settings file is missing:
- `sync.autoFetch = false`
- `sync.fetchIntervalMinutes = 15`
- `sync.pullStrategy = "merge"`
- `commit.defaultBranch = "main"`
- `commit.signingEnabled = false`
- `commit.lineEndings = "lf"`

## Validation Rules

- `identity.name`: non-empty
- `identity.email`: non-empty and basic email format
- `sync.fetchIntervalMinutes`: one of `5|15|30|60`
- `sync.pullStrategy`: one of `merge|rebase|fast-forward`
- `commit.defaultBranch`: non-empty
- `commit.lineEndings`: one of `lf|crlf|auto`

## Data Flow

### GET `/settings`
1. Resolve repository by `id`.
2. Read `.git/config` identity from git service.
3. Read remotes from git service.
4. Read app settings from JSON file, fallback to defaults.
5. Merge and return combined response.

### PUT `/settings/identity`
1. Resolve repository by `id`.
2. Validate payload.
3. Write `user.name` and `user.email` to `.git/config` through git service.
4. Return `204`.

### PUT `/settings/sync`
1. Resolve repository by `id`.
2. Validate enum fields.
3. Load existing file or defaults.
4. Update `sync` section.
5. Atomic write (`tmp` + `rename`).
6. Return `204`.

### PUT `/settings/commit`
1. Resolve repository by `id`.
2. Validate enum fields.
3. Load existing file or defaults.
4. Update `commit` section.
5. Atomic write (`tmp` + `rename`).
6. Return `204`.

## Backend Components

### Models
Add settings models in `server/internal/models/repository.go`:
- `RepoSettings`
- `RepoSettingsIdentity`
- `RepoSettingsSync`
- `RepoSettingsCommit`
- `RepoRemote`
- payload structs for each PUT

### Git service (`server/internal/git/service.go`)
Add:
- `SetGitConfigIdentity(repoPath, name, email) error`
- `GetRemotes(repoPath) ([]models.RepoRemote, error)`

### Handler (`server/internal/api/handlers/repository.go`)
Add:
- `GetRepoSettings`
- `UpdateRepoSettingsIdentity`
- `UpdateRepoSettingsSync`
- `UpdateRepoSettingsCommit`

Add private helpers for:
- defaults
- file load/save
- payload validation

### Routes (`server/internal/api/routes.go`)
Register new settings routes under `/api/repos/{id}`.

## Error Handling

- `404`: repo not found
- `400`: invalid payload
- `500`: read/write/git command failures

Response body uses existing `http.Error` style for consistency.

## Tests

Add/extend tests:
- `server/internal/api/handlers/repository_test.go`
  - GET settings success
  - PUT identity/sync/commit success
  - validation failures
  - repo missing
- `server/internal/git/service_test.go`
  - set identity writes config
  - get remotes reads configured remotes

## Out of Scope

- Background auto-fetch scheduler behavior
- Applying commit sync settings to git CLI commands globally
- Migration/versioning beyond defaults-based bootstrap
