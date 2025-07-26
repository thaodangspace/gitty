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