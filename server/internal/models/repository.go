package models

import (
	"time"
)

type Repository struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	Path        string    `json:"path"`
	URL         string    `json:"url,omitempty"`
	Description string    `json:"description,omitempty"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
	IsLocal     bool      `json:"is_local"`
	CurrentBranch string  `json:"current_branch,omitempty"`
}

type RepositoryStatus struct {
	RepositoryID string            `json:"repository_id"`
	Branch       string            `json:"branch"`
	IsClean      bool              `json:"is_clean"`
	Ahead        int               `json:"ahead"`
	Behind       int               `json:"behind"`
	Staged       []FileChange      `json:"staged"`
	Modified     []FileChange      `json:"modified"`
	Untracked    []string          `json:"untracked"`
	Conflicts    []string          `json:"conflicts"`
}

type FileChange struct {
	Path   string `json:"path"`
	Status string `json:"status"`
	Type   string `json:"type"`
}

type Branch struct {
	Name      string `json:"name"`
	IsCurrent bool   `json:"is_current"`
	IsRemote  bool   `json:"is_remote"`
	Upstream  string `json:"upstream,omitempty"`
	LastCommit *Commit `json:"last_commit,omitempty"`
}

type Commit struct {
	Hash      string    `json:"hash"`
	Message   string    `json:"message"`
	Author    Author    `json:"author"`
	Date      time.Time `json:"date"`
	ParentHash string   `json:"parent_hash,omitempty"`
}

type Author struct {
	Name  string `json:"name"`
	Email string `json:"email"`
}

type FileInfo struct {
	Path        string    `json:"path"`
	Name        string    `json:"name"`
	IsDirectory bool      `json:"is_directory"`
	Size        int64     `json:"size"`
	ModTime     time.Time `json:"mod_time"`
	Mode        string    `json:"mode"`
}

type CommitRequest struct {
	Message string   `json:"message"`
	Files   []string `json:"files"`
	Author  Author   `json:"author,omitempty"`
}

type CreateRepositoryRequest struct {
	Name        string `json:"name"`
	Path        string `json:"path,omitempty"`
	URL         string `json:"url,omitempty"`
	Description string `json:"description,omitempty"`
	IsLocal     bool   `json:"is_local"`
}

type DiffResult struct {
	OldFile string     `json:"old_file"`
	NewFile string     `json:"new_file"`
	Hunks   []DiffHunk `json:"hunks"`
}

type DiffHunk struct {
	OldStart int        `json:"old_start"`
	OldCount int        `json:"old_count"`
	NewStart int        `json:"new_start"`
	NewCount int        `json:"new_count"`
	Lines    []DiffLine `json:"lines"`
}

type DiffLine struct {
	Type    string `json:"type"`
	Content string `json:"content"`
	OldLine int    `json:"old_line,omitempty"`
	NewLine int    `json:"new_line,omitempty"`
}

type DirectoryEntry struct {
	Name        string `json:"name"`
	Path        string `json:"path"`
	IsDirectory bool   `json:"is_directory"`
	IsHidden    bool   `json:"is_hidden"`
	Size        int64  `json:"size,omitempty"`
	ModTime     string `json:"mod_time,omitempty"`
	Permissions string `json:"permissions,omitempty"`
	IsGitRepo   bool   `json:"is_git_repo"`
}

type DirectoryListing struct {
	CurrentPath string           `json:"current_path"`
	ParentPath  string           `json:"parent_path,omitempty"`
	Entries     []DirectoryEntry `json:"entries"`
	CanGoUp     bool             `json:"can_go_up"`
}

// RepoDirectoryListing is the response for browsing a repository directory
// Unlike DirectoryListing (for filesystem), this uses FileInfo and supports pagination
type RepoDirectoryListing struct {
	Path       string     `json:"path"`        // Current directory path (empty for root)
	ParentPath string     `json:"parent_path"` // Parent directory (empty if root)
	Entries    []FileInfo `json:"entries"`     // Files and directories in current path
	TotalCount int        `json:"total_count"` // Total entries in directory (before pagination)
	HasMore    bool       `json:"has_more"`    // True if more entries exist
	Offset     int        `json:"offset"`      // Current pagination offset
}

type CommitDetail struct {
	Hash       string     `json:"hash"`
	Message    string     `json:"message"`
	Author     Author     `json:"author"`
	Date       time.Time  `json:"date"`
	ParentHash string     `json:"parent_hash,omitempty"`
	Changes    []FileDiff `json:"changes"`
	Stats      DiffStats  `json:"stats"`
}

type FileDiff struct {
	Path       string `json:"path"`
	ChangeType string `json:"change_type"`
	Additions  int    `json:"additions"`
	Deletions  int    `json:"deletions"`
	Patch      string `json:"patch"`
}

type DiffStats struct {
	Additions    int `json:"additions"`
	Deletions    int `json:"deletions"`
	FilesChanged int `json:"files_changed"`
}

type GenerateCommitMessageResponse struct {
	Message string `json:"message"`
}

type GitConfig struct {
	Name  string `json:"name"`
	Email string `json:"email"`
}

// ─── TOKENIZED DIFF MODELS ───
// Used for syntax-highlighted diffs sent to the mobile client

// Token - a syntax-highlighted text fragment
type Token struct {
	Text  string `json:"text"`
	Color string `json:"color"`
}

// DiffLineTokenized - single line in a diff hunk
type DiffLineTokenized struct {
	Type   string  `json:"type"`   // "added" | "deleted" | "context"
	Tokens []Token `json:"tokens"` // syntax-highlighted fragments
	OldNum int     `json:"oldNum,omitempty"`
	NewNum int     `json:"newNum,omitempty"`
}

// DiffBlock - a group of consecutive lines of the same type
type DiffBlock struct {
	Type      string              `json:"type"`      // "added" | "deleted" | "context"
	Lines     []DiffLineTokenized `json:"lines"`     // all lines in this block
	StartOld  int                 `json:"startOld"`  // first old line number (0 if N/A)
	EndOld    int                 `json:"endOld"`    // last old line number (0 if N/A)
	StartNew  int                 `json:"startNew"`  // first new line number (0 if N/A)
	EndNew    int                 `json:"endNew"`    // last new line number (0 if N/A)
	Collapsed bool                `json:"collapsed"` // true if context block >= 6 lines
}

// DiffHunkTokenized - contiguous section of changed lines
type DiffHunkTokenized struct {
	Header string      `json:"header"` // "@@ -14,8 +14,10 @@"
	Blocks []DiffBlock `json:"blocks"` // grouped lines
}

// TokenizedDiff - complete tokenized diff for a single file
type TokenizedDiff struct {
	Filename   string              `json:"filename"`
	Hunks      []DiffHunkTokenized `json:"hunks"`
	Additions  int                 `json:"additions"`
	Deletions  int                 `json:"deletions"`
	HasMore    bool                `json:"has_more"`
	NextCursor int                 `json:"next_cursor,omitempty"`
	TotalHunks int                 `json:"total_hunks"`
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