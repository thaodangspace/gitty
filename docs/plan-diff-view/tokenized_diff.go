package models

import "time"

// ─── TOKENIZED DIFF MODELS ───
// These are the structures sent to the React Native client.
// The client renders them directly — no parsing or highlighting needed.

// Token represents a syntax-highlighted text fragment.
type Token struct {
	Text  string `json:"text"`
	Color string `json:"color"`
}

// DiffLine is a single line in a diff hunk.
type DiffLine struct {
	Type   string  `json:"type"`   // "added" | "deleted" | "context"
	Tokens []Token `json:"tokens"` // syntax-highlighted fragments
	OldNum int     `json:"oldNum,omitempty"` // line number in old file
	NewNum int     `json:"newNum,omitempty"` // line number in new file
}

// DiffHunk is a contiguous section of changed lines.
type DiffHunk struct {
	Header string     `json:"header"` // e.g. "@@ -14,8 +14,10 @@"
	Lines  []DiffLine `json:"lines"`
}

// TokenizedDiff is the complete tokenized diff for a single file.
type TokenizedDiff struct {
	Filename  string     `json:"filename"`
	Hunks     []DiffHunk `json:"hunks"`
	Additions int        `json:"additions"`
	Deletions int        `json:"deletions"`
}

// TokenizedFileDiff wraps a tokenized diff with file metadata.
type TokenizedFileDiff struct {
	Path       string        `json:"path"`
	ChangeType string        `json:"changeType"` // "added" | "modified" | "deleted"
	Diff       TokenizedDiff `json:"diff"`
}

// TokenizedCommitDiff is the full tokenized diff for a commit.
type TokenizedCommitDiff struct {
	Hash    string              `json:"hash"`
	Message string              `json:"message"`
	Author  Author              `json:"author"`
	Date    time.Time           `json:"date"`
	Files   []TokenizedFileDiff `json:"files"`
	Stats   DiffStats           `json:"stats"`
}
