# Server-Side Diff Tokenization - Implementation Plan

## Context

The mobile Git client needs syntax-highlighted diffs. Currently, the server returns raw unified diff text, and doing syntax highlighting on the React Native client is problematic (bundles large JS tokenizer, blocks JS thread, causes jank on large diffs).

**Solution**: Move all syntax highlighting to the Go backend using Chroma. The server will parse diffs, tokenize code, and send pre-colored token arrays. The RN client becomes a dumb renderer.

This plan covers the **server-side implementation only**.

## Problem Summary

- Current: Server sends raw diff text → RN client parses and highlights
- Problem: Client-side highlighting adds ~200KB+ bundle size, blocks JS thread, causes jank on 500+ line diffs
- Solution: Server tokenizes with Chroma → sends JSON with colored tokens → RN renders `<Text style={{color: token.color}}>`

## Architecture Overview

```
Server: git diff → parseDiffContent() → tokenizeFullSource() (Chroma) → JSON response
Client: TanStack Query caches → TokenizedDiffView renders tokens directly
```

## Data Models (JSON Contract)

The server will add these models to `server/internal/models/repository.go`:

```go
// Token - a syntax-highlighted text fragment
type Token struct {
    Text  string `json:"text"`
    Color string `json:"color"`
}

// DiffLine - single line in a diff hunk
type DiffLine struct {
    Type   string  `json:"type"`   // "added" | "deleted" | "context"
    Tokens []Token `json:"tokens"` // syntax-highlighted fragments
    OldNum int     `json:"oldNum,omitempty"`
    NewNum int     `json:"newNum,omitempty"`
}

// DiffHunk - contiguous section of changed lines
type DiffHunk struct {
    Header string     `json:"header"` // "@@ -14,8 +14,10 @@"
    Lines  []DiffLine `json:"lines"`
}

// TokenizedDiff - complete tokenized diff for a single file
type TokenizedDiff struct {
    Filename  string     `json:"filename"`
    Hunks     []DiffHunk `json:"hunks"`
    Additions int        `json:"additions"`
    Deletions int        `json:"deletions"`
}

// TokenizedFileDiff - wraps tokenized diff with file metadata
type TokenizedFileDiff struct {
    Path       string        `json:"path"`
    ChangeType string        `json:"changeType"` // "added" | "modified" | "deleted"
    Diff       TokenizedDiff `json:"diff"`
}

// TokenizedCommitDiff - full tokenized diff for a commit
type TokenizedCommitDiff struct {
    Hash    string              `json:"hash"`
    Message string              `json:"message"`
    Author  Author              `json:"author"`
    Date    time.Time           `json:"date"`
    Files   []TokenizedFileDiff `json:"files"`
    Stats   DiffStats           `json:"stats"`
}
```

## Files to Create

### 1. `server/internal/git/diff_tokenizer.go`

Core tokenization logic containing:
- **Dark theme map** - 70+ syntax color mappings (purple keywords, green strings, blue functions, etc.)
- **`lexerForFile()`** - Detect language by filename/extension (TypeScript, Go, Python, Rust, etc.)
- **`colorForToken()`** - Resolve token type to hex color (walks up type hierarchy for fallbacks)
- **`tokenizeLine()`** - Single-line tokenization (fallback mode)
- **`tokenizeFullSource()`** - Full-file tokenization for multi-line accuracy
- **`parseDiffContent()`** - Parse unified diff into structured lines with line numbers
- **`TokenizeDiff()`** - Main public API: diff text → `TokenizedDiff`
- **`TokenizeDiffFromPatch()`** - Convenience wrapper using existing `GetFileDiff`/`GetStagedDiff`
- **`TokenizeCommitDiff()`** - Tokenize all files in a commit

**Key algorithm insight**: Reconstruct "old" and "new" virtual source files from the diff, tokenize each fully (for multi-line string/comment accuracy), then map tokens back to original diff line indices.

### 2. `server/internal/api/handlers/diff.go`

HTTP handlers for the new API endpoints:
- **`HandleFileDiff`** - `GET /api/repos/{id}/diff/tokenized?path=<file>&staged=<bool>`
- **`HandleCommitDiff`** - `GET /api/repos/{id}/diff/commit/tokenized?hash=<commit>`

Both handlers:
1. Extract query params
2. Call git service methods
3. Return JSON response
4. Handle errors with appropriate HTTP status codes

## Files to Modify

### 1. `server/internal/models/repository.go`

Add the tokenized diff model structs (Token, DiffLine, DiffHunk, TokenizedDiff, TokenizedFileDiff, TokenizedCommitDiff) at the end of the file.

### 2. `server/internal/git/service.go`

The tokenization methods will be added to the existing `Service` struct in a separate file (`diff_tokenizer.go`), but they depend on existing methods:
- `GetFileDiff()` - already exists
- `GetStagedDiff()` - already exists
- `GetCommitDetails()` - already exists

No modifications needed to this file - the tokenizer will compose with existing methods.

### 3. `server/internal/api/routes.go`

Register the new diff handler and routes:

```go
diffHandler := api.NewDiffHandler(gitService)
// Inside /repos/{id} route:
r.Get("/diff/tokenized/*", diffHandler.HandleFileDiff)
r.Get("/diff/commit/tokenized", diffHandler.HandleCommitDiff)
```

### 4. `server/go.mod`

Add Chroma dependency:
```bash
go get github.com/alecthomas/chroma/v2
```

## Implementation Steps

1. **Add Chroma dependency**
   ```bash
   cd server && go get github.com/alecthomas/chroma/v2
   ```

2. **Create `server/internal/models/tokenized_diff.go`** (or add to repository.go)
   - Add all tokenized diff structs with JSON tags

3. **Create `server/internal/git/diff_tokenizer.go`**
   - Copy implementation from `docs/plan-diff-view/diff_tokenizer.go`
   - Ensure package name matches (`git`)
   - Verify imports point to correct module paths

4. **Create `server/internal/api/handlers/diff.go`**
   - NewDiffHandler constructor
   - HandleFileDiff method
   - HandleCommitDiff method

5. **Register routes in `server/internal/api/routes.go`**
   - Import handlers package
   - Create diff handler instance
   - Add route handlers

6. **Test endpoints manually**
   - File diff: `GET /api/repos/{id}/diff/tokenized?path=main.go`
   - Commit diff: `GET /api/repos/{id}/diff/commit/tokenized?hash=abc123`

## Critical File Paths

| File | Purpose |
|------|---------|
| `server/internal/models/repository.go` | Add tokenized diff model structs |
| `server/internal/git/diff_tokenizer.go` | Core tokenization algorithm (Chroma lexer, diff parsing, assembly) |
| `server/internal/api/handlers/diff.go` | HTTP handlers for `/api/diff/*` endpoints |
| `server/internal/api/routes.go` | Register new diff routes |
| `server/go.mod` | Add Chroma v2 dependency |

## API Endpoints

### GET /api/repos/{id}/diff/tokenized

Returns tokenized diff for a single file.

**Query params:**
- `path` - relative file path (required)
- `staged` - "true" for staged diff, omit for working tree (optional, default false)

**Response:** `TokenizedDiff` JSON

### GET /api/repos/{id}/diff/commit/tokenized

Returns tokenized diffs for all files in a commit.

**Query params:**
- `hash` - commit SHA (required)

**Response:** `TokenizedCommitDiff` JSON

## Verification

1. **Build test**: `go build ./server/...` compiles without errors
2. **Unit tests**: Run existing tests to ensure no regressions
3. **Manual API test**:
   - Start server
   - Make GET request to `/api/repos/{id}/diff/tokenized?path=somefile.go`
   - Verify response contains `hunks`, each with `lines` containing `tokens` arrays
   - Each token has `text` and `color` fields
4. **Commit diff test**: Request commit tokenized diff, verify `files` array contains multiple `TokenizedFileDiff` objects

## Design Decisions

1. **Full-file tokenization**: We reconstruct complete old/new source files and tokenize them fully (not line-by-line). This ensures multi-line strings/comments are highlighted correctly. Tradeoff: slightly more complex algorithm, but much better accuracy.

2. **Baked-in colors**: Tokens include hex colors directly rather than token-type enums. This eliminates theme engine on client, simplifies renderer. Tradeoff: ~10-15% larger payloads, but eliminates client-side complexity.

3. **Dark theme only**: Initial implementation supports dark mode only. To add light mode later: add `theme` query param and second color map on server.

4. **Existing diff methods reused**: `TokenizeDiffFromPatch` calls existing `GetFileDiff`/`GetStagedDiff` - no duplicate diff generation logic.

## Out of Scope (Client Implementation)

This plan covers server-side only. The client implementation will require:
- TypeScript types in `www/src/types/api.ts` (matching Go models)
- API client methods in `www/src/lib/api-client.ts`
- React hooks with TanStack Query
- UI components to render tokenized diffs

These will be planned separately after server implementation is complete.
