package models

import (
	"time"
)

type Repository struct {
	ID            string    `json:"id" example:"repo-abc123"`
	Name          string    `json:"name" example:"my-project"`
	Path          string    `json:"path" example:"/Users/john/.gitty/repositories/my-project"`
	URL           string    `json:"url,omitempty" example:"https://github.com/user/repo.git"`
	Description   string    `json:"description,omitempty" example:"My awesome project"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
	IsLocal       bool      `json:"is_local" example:"true"`
	CurrentBranch string    `json:"current_branch,omitempty" example:"main"`
}

type RepositoryStatus struct {
	RepositoryID string       `json:"repository_id" example:"repo-abc123"`
	Branch       string       `json:"branch" example:"main"`
	IsClean      bool         `json:"is_clean" example:"false"`
	Ahead        int          `json:"ahead" example:"2"`
	Behind       int          `json:"behind" example:"1"`
	Staged       []FileChange `json:"staged"`
	Modified     []FileChange `json:"modified"`
	Untracked    []string     `json:"untracked"`
	Conflicts    []string     `json:"conflicts"`
}

type FileChange struct {
	Path   string `json:"path" example:"src/main.go"`
	Status string `json:"status" example:"modified"`
	Type   string `json:"type" example:"A"`
}

type Branch struct {
	Name       string  `json:"name" example:"feature/new-feature"`
	IsCurrent  bool    `json:"is_current" example:"true"`
	IsRemote   bool    `json:"is_remote" example:"false"`
	Upstream   string  `json:"upstream,omitempty" example:"origin/main"`
	LastCommit *Commit `json:"last_commit,omitempty"`
}

type Commit struct {
	Hash       string    `json:"hash" example:"abc123def456..."`
	Message    string    `json:"message" example:"Fix bug in authentication"`
	Author     Author    `json:"author"`
	Date       time.Time `json:"date"`
	ParentHash string    `json:"parent_hash,omitempty"`
}

type Author struct {
	Name  string `json:"name" example:"John Doe"`
	Email string `json:"email" example:"john@example.com"`
}

type FileInfo struct {
	Path        string    `json:"path" example:"src/main.go"`
	Name        string    `json:"name" example:"main.go"`
	IsDirectory bool      `json:"is_directory" example:"false"`
	Size        int64     `json:"size" example:"1024"`
	ModTime     time.Time `json:"mod_time"`
	Mode        string    `json:"mode"`
}

type CommitRequest struct {
	Message string   `json:"message" example:"Fix typo in README"`
	Files   []string `json:"files"`
	Author  Author   `json:"author,omitempty"`
}

type CreateRepositoryRequest struct {
	Name        string `json:"name" example:"my-new-repo"`
	Path        string `json:"path,omitempty" example:"/home/user/projects/my-repo"`
	URL         string `json:"url,omitempty" example:"https://github.com/user/repo.git"`
	Description string `json:"description,omitempty" example:"A new repository"`
	IsLocal     bool   `json:"is_local" example:"true"`
}

type DiffResult struct {
	OldFile string     `json:"old_file" example:"src/old.go"`
	NewFile string     `json:"new_file" example:"src/new.go"`
	Hunks   []DiffHunk `json:"hunks"`
}

type DiffHunk struct {
	OldStart int        `json:"old_start" example:"10"`
	OldCount int        `json:"old_count" example:"5"`
	NewStart int        `json:"new_start" example:"12"`
	NewCount int        `json:"new_count" example:"7"`
	Lines    []DiffLine `json:"lines"`
}

type DiffLine struct {
	Type    string `json:"type" example:"added"`
	Content string `json:"content" example:"func main() {}"`
	OldLine int    `json:"old_line,omitempty" example:"0"`
	NewLine int    `json:"new_line,omitempty" example:"1"`
}

type DirectoryEntry struct {
	Name        string `json:"name" example:"my-project"`
	Path        string `json:"path" example:"/home/user/projects/my-project"`
	IsDirectory bool   `json:"is_directory" example:"true"`
	IsHidden    bool   `json:"is_hidden" example:"false"`
	Size        int64  `json:"size,omitempty" example:"4096"`
	ModTime     string `json:"mod_time,omitempty" example:"2024-01-15T10:30:00Z"`
	Permissions string `json:"permissions,omitempty" example:"drwxr-xr-x"`
	IsGitRepo   bool   `json:"is_git_repo" example:"true"`
}

type DirectoryListing struct {
	CurrentPath string           `json:"current_path" example:"/home/user/projects"`
	ParentPath  string           `json:"parent_path,omitempty" example:"/home/user"`
	Entries     []DirectoryEntry `json:"entries"`
	CanGoUp     bool             `json:"can_go_up" example:"true"`
}

// RepoDirectoryListing is the response for browsing a repository directory
// Unlike DirectoryListing (for filesystem), this uses FileInfo and supports pagination
type RepoDirectoryListing struct {
	Path       string     `json:"path" example:"src/components"`        // Current directory path (empty for root)
	ParentPath string     `json:"parent_path" example:"src"`              // Parent directory (empty if root)
	Entries    []FileInfo `json:"entries"`                                  // Files and directories in current path
	TotalCount int        `json:"total_count" example:"42"`               // Total entries in directory (before pagination)
	HasMore    bool       `json:"has_more" example:"true"`                // True if more entries exist
	Offset     int        `json:"offset" example:"0"`                     // Current pagination offset
}

type CommitDetail struct {
	Hash       string     `json:"hash" example:"abc123def456..."`
	Message    string     `json:"message" example:"Fix bug in authentication"`
	Author     Author     `json:"author"`
	Date       time.Time  `json:"date"`
	ParentHash string     `json:"parent_hash,omitempty"`
	Changes    []FileDiff `json:"changes"`
	Stats      DiffStats  `json:"stats"`
}

type FileDiff struct {
	Path       string `json:"path" example:"src/auth.go"`
	ChangeType string `json:"change_type" example:"modified"`
	Additions  int    `json:"additions" example:"5"`
	Deletions  int    `json:"deletions" example:"2"`
	Patch      string `json:"patch"`
}

type DiffStats struct {
	Additions    int `json:"additions" example:"10"`
	Deletions    int `json:"deletions" example:"3"`
	FilesChanged int `json:"files_changed" example:"2"`
}

type GenerateCommitMessageResponse struct {
	Message string `json:"message" example:"Fix: Handle null pointer in auth flow"`
}

type GitConfig struct {
	Name  string `json:"name" example:"John Doe"`
	Email string `json:"email" example:"john@example.com"`
}

type RepoRemote struct {
	Name string `json:"name" example:"origin"`
	URL  string `json:"url" example:"https://github.com/user/repo.git"`
}

type RepoIdentitySettings struct {
	Name  string `json:"name" example:"John Doe"`
	Email string `json:"email" example:"john@example.com"`
}

type RepoSyncSettings struct {
	AutoFetch            bool   `json:"autoFetch" example:"true"`
	FetchIntervalMinutes int    `json:"fetchIntervalMinutes" example:"15"`
	PullStrategy         string `json:"pullStrategy" example:"merge"`
}

type RepoCommitSettings struct {
	DefaultBranch  string `json:"defaultBranch" example:"main"`
	SigningEnabled bool   `json:"signingEnabled" example:"false"`
	LineEndings    string `json:"lineEndings" example:"lf"`
}

type RepoSettings struct {
	Identity RepoIdentitySettings `json:"identity"`
	Sync     RepoSyncSettings     `json:"sync"`
	Commit   RepoCommitSettings   `json:"commit"`
	Remotes  []RepoRemote         `json:"remotes"`
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
