package api

import (
	"encoding/json"
	"net/http"

	"gitweb/server/internal/git"
)

type DiffHandler struct {
	gitService *git.Service
}

func NewDiffHandler(gitService *git.Service) *DiffHandler {
	return &DiffHandler{gitService: gitService}
}

// GET /api/repos/{repoPath}/diff/file?path=<filePath>&staged=<bool>
// Returns tokenized diff for a single working-tree or staged file.
func (h *DiffHandler) HandleFileDiff(w http.ResponseWriter, r *http.Request) {
	repoPath := r.URL.Query().Get("repo")
	filePath := r.URL.Query().Get("path")
	staged := r.URL.Query().Get("staged") == "true"

	if repoPath == "" || filePath == "" {
		http.Error(w, `{"error":"repo and path are required"}`, http.StatusBadRequest)
		return
	}

	result, err := h.gitService.TokenizeDiffFromPatch(repoPath, filePath, staged)
	if err != nil {
		http.Error(w, `{"error":"`+err.Error()+`"}`, http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}

// GET /api/repos/{repoPath}/diff/commit?hash=<commitHash>
// Returns tokenized diffs for all files in a commit.
func (h *DiffHandler) HandleCommitDiff(w http.ResponseWriter, r *http.Request) {
	repoPath := r.URL.Query().Get("repo")
	commitHash := r.URL.Query().Get("hash")

	if repoPath == "" || commitHash == "" {
		http.Error(w, `{"error":"repo and hash are required"}`, http.StatusBadRequest)
		return
	}

	result, err := h.gitService.TokenizeCommitDiff(repoPath, commitHash)
	if err != nil {
		http.Error(w, `{"error":"`+err.Error()+`"}`, http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}
