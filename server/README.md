# GitWeb Backend

GitWeb's backend is a Go service that exposes a REST API for managing Git repositories and browsing the local filesystem.

## Features
- **Repository management**: list, create, import, and delete repositories
- **Branch & commit operations**: view commit history, create commits, manage branches, and inspect commit details
- **File management**: browse repository trees, read or update file contents, view diffs, and stage or unstage changes
- **Remote sync**: push to and pull from remotes
- **Filesystem browsing**: explore directories on the host machine (restricted to the user home directory)

## Architecture
The entry point is `cmd/gitweb/main.go`. The server sets up a repository data path, registers routes, and starts an HTTP server on the configured port. Routes are organized under `/api` and use Chi with CORS, logging, and recovery middleware.

Major internal packages:
- `internal/api`: HTTP routes and handlers for repositories and filesystem operations
- `internal/git`: wrapper around `go-git` for repository actions
- `internal/filesystem`: directory listing utilities with path restrictions
- `internal/models`: shared data structures for API responses

## Running Locally
```bash
cd server
GO111MODULE=on go run ./cmd/gitweb
```
Environment variables:
- `GITWEB_DATA_PATH`: directory to store repositories (defaults to `~/.gitweb/repositories`)
- `PORT`: HTTP port (defaults to `8083`)

## API Overview
- `GET /health` – service health check
- `GET /api/repos` – list repositories
- `POST /api/repos` – create a repository or clone from URL
- `POST /api/repos/import` – import an existing repository from disk
- `GET /api/repos/{id}/status` – repository status
- `GET /api/repos/{id}/commits` – commit history
- `GET /api/repos/{id}/branches` – list branches
- `POST /api/repos/{id}/commit` – create commit
- `POST /api/repos/{id}/branches` – create branch
- `PUT /api/repos/{id}/branches/{branch}` – switch branch
- `DELETE /api/repos/{id}/branches/{branch}` – delete branch
- `GET /api/repos/{id}/files` – file tree
- `GET/PUT /api/repos/{id}/files/*` – read or write file
- `GET /api/repos/{id}/diff/*` – file diff
- `POST/DELETE /api/repos/{id}/stage/*` – stage or unstage file
- `POST /api/repos/{id}/push` – push to remote
- `POST /api/repos/{id}/pull` – pull from remote
- `GET /api/filesystem/browse` – browse a directory
- `GET /api/filesystem/roots` – list allowed root paths

## Tests
Run unit tests for all packages:
```bash
go test ./...
```

