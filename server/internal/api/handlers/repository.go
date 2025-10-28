package handlers

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"time"

	"gitweb/server/internal/git"
	"gitweb/server/internal/models"

	"github.com/go-chi/chi/v5"
)

type RepositoryHandler struct {
	gitService   *git.Service
	repositories map[string]*models.Repository
	dataPath     string
	watcher      *git.RepositoryWatcher
}

func NewRepositoryHandler(dataPath string) *RepositoryHandler {
	watcher, err := git.NewRepositoryWatcher()
	if err != nil {
		// Log error but don't fail initialization - we can still function without watching
		fmt.Printf("Warning: Failed to initialize repository watcher: %v\n", err)
	}

	return &RepositoryHandler{
		gitService:   git.NewService(),
		repositories: make(map[string]*models.Repository),
		dataPath:     dataPath,
		watcher:      watcher,
	}
}

func (h *RepositoryHandler) loadRepositories() error {
	if _, err := os.Stat(h.dataPath); os.IsNotExist(err) {
		return os.MkdirAll(h.dataPath, 0755)
	}

	entries, err := os.ReadDir(h.dataPath)
	if err != nil {
		return err
	}

	for _, entry := range entries {
		if entry.IsDir() {
			repoPath := filepath.Join(h.dataPath, entry.Name())
			if h.isGitRepository(repoPath) {
				repo := &models.Repository{
					ID:        entry.Name(),
					Name:      entry.Name(),
					Path:      repoPath,
					IsLocal:   true,
					CreatedAt: time.Now(),
					UpdatedAt: time.Now(),
				}
				h.repositories[repo.ID] = repo
			}
		}
	}

	return nil
}

func (h *RepositoryHandler) isGitRepository(path string) bool {
	gitPath := filepath.Join(path, ".git")
	_, err := os.Stat(gitPath)
	return err == nil
}

func (h *RepositoryHandler) ListRepositories(w http.ResponseWriter, r *http.Request) {
	if err := h.loadRepositories(); err != nil {
		http.Error(w, fmt.Sprintf("Failed to load repositories: %v", err), http.StatusInternalServerError)
		return
	}

	repos := make([]*models.Repository, 0, len(h.repositories))
	for _, repo := range h.repositories {
		status, err := h.gitService.GetRepositoryStatus(repo.Path)
		if err == nil {
			repo.CurrentBranch = status.Branch
		}
		repos = append(repos, repo)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(repos)
}

func (h *RepositoryHandler) CreateRepository(w http.ResponseWriter, r *http.Request) {
	var req models.CreateRepositoryRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if req.Name == "" {
		http.Error(w, "Repository name is required", http.StatusBadRequest)
		return
	}

	repoID := req.Name
	repoPath := filepath.Join(h.dataPath, req.Name)

	if _, err := os.Stat(repoPath); !os.IsNotExist(err) {
		http.Error(w, "Repository already exists", http.StatusConflict)
		return
	}

	var err error
	
	if req.URL != "" {
		_, err = h.gitService.CloneRepository(req.URL, repoPath)
		if err != nil {
			http.Error(w, fmt.Sprintf("Failed to clone repository: %v", err), http.StatusInternalServerError)
			return
		}
	} else {
		_, err = h.gitService.InitRepository(repoPath)
		if err != nil {
			http.Error(w, fmt.Sprintf("Failed to initialize repository: %v", err), http.StatusInternalServerError)
			return
		}
	}

	repo := &models.Repository{
		ID:          repoID,
		Name:        req.Name,
		Path:        repoPath,
		URL:         req.URL,
		Description: req.Description,
		IsLocal:     req.URL == "",
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	h.repositories[repoID] = repo

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(repo)
}

func (h *RepositoryHandler) GetRepository(w http.ResponseWriter, r *http.Request) {
	repoID := chi.URLParam(r, "id")
	
	repo, exists := h.repositories[repoID]
	if !exists {
		http.Error(w, "Repository not found", http.StatusNotFound)
		return
	}

	status, err := h.gitService.GetRepositoryStatus(repo.Path)
	if err == nil {
		repo.CurrentBranch = status.Branch
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(repo)
}

func (h *RepositoryHandler) DeleteRepository(w http.ResponseWriter, r *http.Request) {
	repoID := chi.URLParam(r, "id")
	
	repo, exists := h.repositories[repoID]
	if !exists {
		http.Error(w, "Repository not found", http.StatusNotFound)
		return
	}

	if err := os.RemoveAll(repo.Path); err != nil {
		http.Error(w, fmt.Sprintf("Failed to delete repository: %v", err), http.StatusInternalServerError)
		return
	}

	delete(h.repositories, repoID)

	w.WriteHeader(http.StatusNoContent)
}

func (h *RepositoryHandler) GetRepositoryStatus(w http.ResponseWriter, r *http.Request) {
	repoID := chi.URLParam(r, "id")

	repo, exists := h.repositories[repoID]
	if !exists {
		http.Error(w, "Repository not found", http.StatusNotFound)
		return
	}

	// Long polling: only wait for changes if "wait" query parameter is present
	// This allows immediate response on first load, and long polling on subsequent requests
	shouldWait := r.URL.Query().Get("wait") == "true"
	if shouldWait && h.watcher != nil {
		// Wait for change notification or timeout (30 seconds)
		h.watcher.WaitForChange(repo.Path, 30*time.Second)
		// Whether we got a change or timeout, continue to return current status
	}

	status, err := h.gitService.GetRepositoryStatus(repo.Path)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to get repository status: %v", err), http.StatusInternalServerError)
		return
	}

	status.RepositoryID = repoID

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(status)
}

func (h *RepositoryHandler) GetCommitHistory(w http.ResponseWriter, r *http.Request) {
	repoID := chi.URLParam(r, "id")
	
	repo, exists := h.repositories[repoID]
	if !exists {
		http.Error(w, "Repository not found", http.StatusNotFound)
		return
	}

	limitStr := r.URL.Query().Get("limit")
	limit := 50
	if limitStr != "" {
		if parsedLimit, err := strconv.Atoi(limitStr); err == nil && parsedLimit > 0 {
			limit = parsedLimit
		}
	}

	commits, err := h.gitService.GetCommitHistory(repo.Path, limit)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to get commit history: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(commits)
}

func (h *RepositoryHandler) GetBranches(w http.ResponseWriter, r *http.Request) {
	repoID := chi.URLParam(r, "id")
	
	repo, exists := h.repositories[repoID]
	if !exists {
		http.Error(w, "Repository not found", http.StatusNotFound)
		return
	}

	branches, err := h.gitService.GetBranches(repo.Path)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to get branches: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(branches)
}

func (h *RepositoryHandler) CreateCommit(w http.ResponseWriter, r *http.Request) {
	repoID := chi.URLParam(r, "id")
	
	repo, exists := h.repositories[repoID]
	if !exists {
		http.Error(w, "Repository not found", http.StatusNotFound)
		return
	}

	var req models.CommitRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if req.Message == "" {
		http.Error(w, "Commit message is required", http.StatusBadRequest)
		return
	}

	err := h.gitService.CreateCommit(repo.Path, req)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to create commit: %v", err), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusCreated)
	w.Write([]byte(`{"message": "Commit created successfully"}`))
}

func (h *RepositoryHandler) CreateBranch(w http.ResponseWriter, r *http.Request) {
	repoID := chi.URLParam(r, "id")
	
	repo, exists := h.repositories[repoID]
	if !exists {
		http.Error(w, "Repository not found", http.StatusNotFound)
		return
	}

	var req struct {
		Name string `json:"name"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if req.Name == "" {
		http.Error(w, "Branch name is required", http.StatusBadRequest)
		return
	}

	err := h.gitService.CreateBranch(repo.Path, req.Name)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to create branch: %v", err), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusCreated)
	w.Write([]byte(`{"message": "Branch created successfully"}`))
}

func (h *RepositoryHandler) SwitchBranch(w http.ResponseWriter, r *http.Request) {
	repoID := chi.URLParam(r, "id")
	branchName := chi.URLParam(r, "branch")
	
	repo, exists := h.repositories[repoID]
	if !exists {
		http.Error(w, "Repository not found", http.StatusNotFound)
		return
	}

	err := h.gitService.SwitchBranch(repo.Path, branchName)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to switch branch: %v", err), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`{"message": "Branch switched successfully"}`))
}

func (h *RepositoryHandler) GetFileTree(w http.ResponseWriter, r *http.Request) {
	repoID := chi.URLParam(r, "id")
	
	repo, exists := h.repositories[repoID]
	if !exists {
		http.Error(w, "Repository not found", http.StatusNotFound)
		return
	}

	files, err := h.gitService.GetFileTree(repo.Path)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to get file tree: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(files)
}

func (h *RepositoryHandler) GetFileContent(w http.ResponseWriter, r *http.Request) {
	repoID := chi.URLParam(r, "id")
	filePath := chi.URLParam(r, "*")
	
	repo, exists := h.repositories[repoID]
	if !exists {
		http.Error(w, "Repository not found", http.StatusNotFound)
		return
	}

	content, err := h.gitService.GetFileContent(repo.Path, filePath)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to get file content: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/plain")
	w.Write(content)
}

func (h *RepositoryHandler) SaveFileContent(w http.ResponseWriter, r *http.Request) {
	repoID := chi.URLParam(r, "id")
	filePath := chi.URLParam(r, "*")
	
	repo, exists := h.repositories[repoID]
	if !exists {
		http.Error(w, "Repository not found", http.StatusNotFound)
		return
	}

	content, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "Failed to read request body", http.StatusBadRequest)
		return
	}

	err = h.gitService.SaveFileContent(repo.Path, filePath, content)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to save file: %v", err), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`{"message": "File saved successfully"}`))
}

func (h *RepositoryHandler) Push(w http.ResponseWriter, r *http.Request) {
	repoID := chi.URLParam(r, "id")

	repo, exists := h.repositories[repoID]
	if !exists {
		http.Error(w, "Repository not found", http.StatusNotFound)
		return
	}

	err := h.gitService.Push(repo.Path)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to push: %v", err), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`{"message": "Push completed successfully"}`))
}

func (h *RepositoryHandler) ForcePush(w http.ResponseWriter, r *http.Request) {
	repoID := chi.URLParam(r, "id")

	repo, exists := h.repositories[repoID]
	if !exists {
		http.Error(w, "Repository not found", http.StatusNotFound)
		return
	}

	err := h.gitService.ForcePush(repo.Path)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to force push: %v", err), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`{"message": "Force push completed successfully"}`))
}

func (h *RepositoryHandler) ImportRepository(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Path string `json:"path"`
		Name string `json:"name,omitempty"`
	}
	
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if req.Path == "" {
		http.Error(w, "Repository path is required", http.StatusBadRequest)
		return
	}

	// Check if path exists and is a Git repository
	if !h.isGitRepository(req.Path) {
		http.Error(w, "Path is not a valid Git repository", http.StatusBadRequest)
		return
	}

	// Use the folder name as repository name if not provided
	repoName := req.Name
	if repoName == "" {
		repoName = filepath.Base(req.Path)
	}

	// Generate a unique ID for the repository
	repoID := repoName
	counter := 1
	for _, exists := h.repositories[repoID]; exists; _, exists = h.repositories[repoID] {
		repoID = fmt.Sprintf("%s-%d", repoName, counter)
		counter++
	}

	// Create repository record
	repo := &models.Repository{
		ID:        repoID,
		Name:      repoName,
		Path:      req.Path,
		IsLocal:   true,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	// Get current branch
	status, err := h.gitService.GetRepositoryStatus(req.Path)
	if err == nil {
		repo.CurrentBranch = status.Branch
	}

	h.repositories[repoID] = repo

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(repo)
}

func (h *RepositoryHandler) Pull(w http.ResponseWriter, r *http.Request) {
	repoID := chi.URLParam(r, "id")
	
	repo, exists := h.repositories[repoID]
	if !exists {
		http.Error(w, "Repository not found", http.StatusNotFound)
		return
	}

	err := h.gitService.Pull(repo.Path)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to pull: %v", err), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`{"message": "Pull completed successfully"}`))
}

func (h *RepositoryHandler) StageFile(w http.ResponseWriter, r *http.Request) {
	repoID := chi.URLParam(r, "id")
	filePath := chi.URLParam(r, "*")
	
	repo, exists := h.repositories[repoID]
	if !exists {
		http.Error(w, "Repository not found", http.StatusNotFound)
		return
	}

	err := h.gitService.StageFile(repo.Path, filePath)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to stage file: %v", err), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`{"message": "File staged successfully"}`))
}

func (h *RepositoryHandler) UnstageFile(w http.ResponseWriter, r *http.Request) {
	repoID := chi.URLParam(r, "id")
	filePath := chi.URLParam(r, "*")
	
	repo, exists := h.repositories[repoID]
	if !exists {
		http.Error(w, "Repository not found", http.StatusNotFound)
		return
	}

	err := h.gitService.UnstageFile(repo.Path, filePath)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to unstage file: %v", err), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`{"message": "File unstaged successfully"}`))
}

func (h *RepositoryHandler) GetCommitDetails(w http.ResponseWriter, r *http.Request) {
	repoID := chi.URLParam(r, "id")
	commitHash := chi.URLParam(r, "hash")
	
	repo, exists := h.repositories[repoID]
	if !exists {
		http.Error(w, "Repository not found", http.StatusNotFound)
		return
	}

	if commitHash == "" {
		http.Error(w, "Commit hash is required", http.StatusBadRequest)
		return
	}

	commitDetail, err := h.gitService.GetCommitDetails(repo.Path, commitHash)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to get commit details: %v", err), http.StatusInternalServerError)
		return
	}

	// Update files changed count
	commitDetail.Stats.FilesChanged = len(commitDetail.Changes)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(commitDetail)
}

func (h *RepositoryHandler) DeleteBranch(w http.ResponseWriter, r *http.Request) {
	repoID := chi.URLParam(r, "id")
	branchName := chi.URLParam(r, "branch")
	
	repo, exists := h.repositories[repoID]
	if !exists {
		http.Error(w, "Repository not found", http.StatusNotFound)
		return
	}

	if branchName == "" {
		http.Error(w, "Branch name is required", http.StatusBadRequest)
		return
	}

	err := h.gitService.DeleteBranch(repo.Path, branchName)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to delete branch: %v", err), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`{"message": "Branch deleted successfully"}`))
}

func (h *RepositoryHandler) GetFileDiff(w http.ResponseWriter, r *http.Request) {
	repoID := chi.URLParam(r, "id")
	filePath := chi.URLParam(r, "*")
	
	repo, exists := h.repositories[repoID]
	if !exists {
		http.Error(w, "Repository not found", http.StatusNotFound)
		return
	}

	if filePath == "" {
		http.Error(w, "File path is required", http.StatusBadRequest)
		return
	}

	diff, err := h.gitService.GetFileDiff(repo.Path, filePath)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to get file diff: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/plain")
	w.Write([]byte(diff))
}