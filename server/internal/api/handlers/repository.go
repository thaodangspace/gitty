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
}

func NewRepositoryHandler(dataPath string) *RepositoryHandler {
	return &RepositoryHandler{
		gitService:   git.NewService(),
		repositories: make(map[string]*models.Repository),
		dataPath:     dataPath,
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