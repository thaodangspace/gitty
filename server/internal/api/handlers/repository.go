package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"runtime/metrics"
	"strconv"
	"strings"
	"sync"
	"time"

	"gitweb/server/internal/config"
	"gitweb/server/internal/git"
	"gitweb/server/internal/models"
	"gitweb/server/internal/registry"
	"gitweb/server/internal/resources"
	"gitweb/server/internal/services"

	"github.com/go-chi/chi/v5"
)

type repoAppSettings struct {
	Sync   models.RepoSyncSettings   `json:"sync"`
	Commit models.RepoCommitSettings `json:"commit"`
}

type pressureTicker interface {
	Chan() <-chan time.Time
	Stop()
}

type realPressureTicker struct {
	*time.Ticker
}

func (t realPressureTicker) Chan() <-chan time.Time {
	return t.C
}

type RepositoryHandler struct {
	mu            sync.RWMutex
	settingsMu    sync.Mutex
	gitService    *git.Service
	repositories  map[string]*models.Repository
	dataPath      string
	watcher       *git.RepositoryWatcher
	config        *config.Config
	claudeService *services.ClaudeService
	registry      *registry.Registry
	governor      *resources.Governor
	retryAfterSec int

	pressureSampler     func() (float64, error)
	newPressureTicker   func(time.Duration) pressureTicker
	pressureMonitorMu   sync.Mutex
	pressureMonitorStop chan struct{}
	pressureMonitorDone chan struct{}
	logf                func(string, ...any)
}

func defaultRepoAppSettings() repoAppSettings {
	return repoAppSettings{
		Sync: models.RepoSyncSettings{
			AutoFetch:            false,
			FetchIntervalMinutes: 15,
			PullStrategy:         "merge",
		},
		Commit: models.RepoCommitSettings{
			DefaultBranch:  "main",
			SigningEnabled: false,
			LineEndings:    "lf",
		},
	}
}

func NewRepositoryHandler(dataPath string, cfg *config.Config, reg *registry.Registry) *RepositoryHandler {
	watcher, err := git.NewRepositoryWatcher()
	if err != nil {
		// Log error but don't fail initialization - we can still function without watching
		fmt.Printf("Warning: Failed to initialize repository watcher: %v\n", err)
	}

	handler := &RepositoryHandler{
		gitService:      git.NewService(),
		repositories:    make(map[string]*models.Repository),
		dataPath:        dataPath,
		watcher:         watcher,
		config:          cfg,
		claudeService:   services.NewClaudeService(cfg),
		registry:        reg,
		governor:        resources.NewGovernor(resources.FromAppConfig(cfg)),
		retryAfterSec:   retryAfterSeconds(cfg),
		pressureSampler: newPressureSampler(cfg),
		newPressureTicker: func(interval time.Duration) pressureTicker {
			return realPressureTicker{Ticker: time.NewTicker(interval)}
		},
		logf: log.Printf,
	}

	// Load repositories at initialization so they're available for all handlers
	if err := handler.loadRepositories(); err != nil {
		fmt.Printf("Warning: Failed to load repositories during initialization: %v\n", err)
	}

	return handler
}

func retryAfterSeconds(cfg *config.Config) int {
	if cfg != nil && cfg.ResourceGovernor != nil && cfg.ResourceGovernor.RetryAfterSeconds > 0 {
		return cfg.ResourceGovernor.RetryAfterSeconds
	}
	return 3
}

func (h *RepositoryHandler) enterExpensiveOrReject(w http.ResponseWriter, r *http.Request) (func(), bool) {
	admission := h.governor.AdmitExpensive()
	if admission.Admitted {
		return admission.Release, true
	}

	h.logf("resource governor rejected request route=%q reason=%q mode=%q", requestRoute(r), admission.Reason, h.governor.Mode())

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Retry-After", strconv.Itoa(h.retryAfterSec))
	w.WriteHeader(http.StatusServiceUnavailable)
	_ = json.NewEncoder(w).Encode(map[string]string{
		"reason": admission.Reason,
	})

	return nil, false
}

func newPressureSampler(cfg *config.Config) func() (float64, error) {
	if cfg == nil || cfg.ResourceGovernor == nil {
		return nil
	}

	limit := cfg.ResourceGovernor.MemoryLimitBytes
	if limit <= 0 {
		return nil
	}

	return func() (float64, error) {
		samples := []metrics.Sample{
			{Name: "/memory/classes/total:bytes"},
			{Name: "/memory/classes/heap/released:bytes"},
		}
		metrics.Read(samples)
		if samples[0].Value.Kind() != metrics.KindUint64 {
			return 0, fmt.Errorf("read memory pressure metric: unexpected kind %v", samples[0].Value.Kind())
		}
		if samples[1].Value.Kind() != metrics.KindUint64 {
			return 0, fmt.Errorf("read heap released metric: unexpected kind %v", samples[1].Value.Kind())
		}

		return computeMemoryPressure(samples[0].Value.Uint64(), samples[1].Value.Uint64(), limit), nil
	}
}

func computeMemoryPressure(total, heapReleased uint64, limit int64) float64 {
	if limit <= 0 || heapReleased >= total {
		return 0
	}

	return float64(total-heapReleased) / float64(limit)
}

func sampleInterval(cfg *config.Config) time.Duration {
	if cfg != nil && cfg.ResourceGovernor != nil && cfg.ResourceGovernor.SampleIntervalMs > 0 {
		return time.Duration(cfg.ResourceGovernor.SampleIntervalMs) * time.Millisecond
	}
	return 500 * time.Millisecond
}

func requestRoute(r *http.Request) string {
	if r == nil {
		return ""
	}

	if routeCtx := chi.RouteContext(r.Context()); routeCtx != nil {
		if pattern := routeCtx.RoutePattern(); pattern != "" {
			return pattern
		}
	}
	if r.URL == nil {
		return ""
	}
	return r.URL.Path
}

func (h *RepositoryHandler) StartPressureMonitor(ctx context.Context) {
	if h == nil || h.governor == nil || h.pressureSampler == nil {
		return
	}
	if ctx == nil {
		ctx = context.Background()
	}
	if h.config != nil && h.config.ResourceGovernor != nil && !h.config.ResourceGovernor.Enabled {
		return
	}

	h.pressureMonitorMu.Lock()
	defer h.pressureMonitorMu.Unlock()

	if h.pressureMonitorStop != nil {
		return
	}

	ticker := h.newPressureTicker(sampleInterval(h.config))
	stopCh := make(chan struct{})
	doneCh := make(chan struct{})
	h.pressureMonitorStop = stopCh
	h.pressureMonitorDone = doneCh

	go func() {
		defer func() {
			ticker.Stop()

			h.pressureMonitorMu.Lock()
			if h.pressureMonitorStop == stopCh {
				h.pressureMonitorStop = nil
			}
			if h.pressureMonitorDone == doneCh {
				h.pressureMonitorDone = nil
			}
			h.pressureMonitorMu.Unlock()

			close(doneCh)
		}()

		for {
			select {
			case <-ctx.Done():
				return
			case <-stopCh:
				return
			case <-ticker.Chan():
				pressure, err := h.pressureSampler()
				if err != nil {
					h.logf("resource governor pressure sample failed err=%v", err)
					continue
				}

				before := h.governor.Mode()
				h.governor.UpdatePressure(pressure)
				after := h.governor.Mode()
				if after != before {
					h.logf("resource governor mode changed mode=%q pressure=%.4f", after, pressure)
				}
			}
		}
	}()
}

func (h *RepositoryHandler) loadRepositories() error {
	if h.registry == nil {
		return nil
	}

	for _, entry := range h.registry.List() {
		if !h.isGitRepository(entry.Path) {
			fmt.Printf("Warning: registry entry %q path %q is not a valid git repository, skipping\n", entry.ID, entry.Path)
			continue
		}

		repo := &models.Repository{
			ID:          entry.ID,
			Name:        entry.Name,
			Path:        entry.Path,
			URL:         entry.URL,
			Description: entry.Description,
			IsLocal:     entry.URL == "",
			CreatedAt:   entry.ImportedAt,
			UpdatedAt:   time.Now(),
		}
		h.repositories[repo.ID] = repo
	}

	return nil
}

func (h *RepositoryHandler) isGitRepository(path string) bool {
	gitPath := filepath.Join(path, ".git")
	_, err := os.Stat(gitPath)
	return err == nil
}

// @Summary      List all repositories
// @Description  Get all registered Git repositories
// @Tags         repositories
// @Produce      json
// @Success      200  {array}   models.Repository
// @Security     BearerAuth
// @Router       /api/repos [get]
func (h *RepositoryHandler) ListRepositories(w http.ResponseWriter, r *http.Request) {
	h.mu.RLock()
	repos := make([]*models.Repository, 0, len(h.repositories))
	for _, repo := range h.repositories {
		repos = append(repos, repo)
	}
	h.mu.RUnlock()

	for _, repo := range repos {
		status, err := h.gitService.GetRepositoryStatus(repo.Path)
		if err == nil {
			repo.CurrentBranch = status.Branch
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(repos)
}

// @Summary      Create a new repository
// @Description  Create a new Git repository or clone from URL
// @Tags         repositories
// @Accept       json
// @Produce      json
// @Param        body  body     models.CreateRepositoryRequest  true  "Request body"
// @Success      201   {object} models.Repository
// @Failure      400   {string} string "Bad request"
// @Security     BearerAuth
// @Router       /api/repos [post]
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

	// Get branch info
	status, err := h.gitService.GetRepositoryStatus(repoPath)
	if err == nil {
		repo.CurrentBranch = status.Branch
	}

	// Persist to registry first
	if h.registry != nil {
		source := "created"
		if req.URL != "" {
			source = "cloned"
		}
		regEntry := registry.Entry{
			ID:          repoID,
			Name:        req.Name,
			Path:        repoPath,
			URL:         req.URL,
			Description: req.Description,
			Source:      source,
			ImportedAt:  time.Now(),
		}
		if err := h.registry.Add(regEntry); err != nil {
			http.Error(w, fmt.Sprintf("Failed to persist repository: %v", err), http.StatusInternalServerError)
			return
		}
	}

	h.mu.Lock()
	h.repositories[repoID] = repo
	h.mu.Unlock()

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(repo)
}

// @Summary      Get repository by ID
// @Description  Get details for a specific repository
// @Tags         repositories
// @Produce      json
// @Param        id    path     string  true  "Repository ID"
// @Success      200   {object} models.Repository
// @Failure      404   {string} string "Repository not found"
// @Security     BearerAuth
// @Router       /api/repos/{id} [get]
func (h *RepositoryHandler) GetRepository(w http.ResponseWriter, r *http.Request) {
	repoID := chi.URLParam(r, "id")

	h.mu.RLock()
	repo, exists := h.repositories[repoID]
	h.mu.RUnlock()

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

// @Summary      Delete a repository
// @Description  Delete a repository from Gittyd
// @Tags         repositories
// @Param        id    path     string  true  "Repository ID"
// @Success      204   {string} string "No content"
// @Failure      404   {string} string "Repository not found"
// @Security     BearerAuth
// @Router       /api/repos/{id} [delete]
func (h *RepositoryHandler) DeleteRepository(w http.ResponseWriter, r *http.Request) {
	repoID := chi.URLParam(r, "id")

	h.mu.RLock()
	_, exists := h.repositories[repoID]
	h.mu.RUnlock()

	if !exists {
		http.Error(w, "Repository not found", http.StatusNotFound)
		return
	}

	if h.registry != nil {
		if err := h.registry.Remove(repoID); err != nil {
			http.Error(w, fmt.Sprintf("Failed to remove repository from registry: %v", err), http.StatusInternalServerError)
			return
		}
	}

	h.mu.Lock()
	delete(h.repositories, repoID)
	h.mu.Unlock()

	w.WriteHeader(http.StatusNoContent)
}

// @Summary      Get repository status
// @Description  Get working directory status (staged, modified, untracked files)
// @Tags         repositories
// @Produce      json
// @Param        id    path     string  true  "Repository ID"
// @Success      200   {object} models.RepositoryStatus
// @Failure      404   {string} string "Repository not found"
// @Security     BearerAuth
// @Router       /api/repos/{id}/status [get]
func (h *RepositoryHandler) GetRepositoryStatus(w http.ResponseWriter, r *http.Request) {
	repoID := chi.URLParam(r, "id")

	h.mu.RLock()
	repo, exists := h.repositories[repoID]
	h.mu.RUnlock()

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

// @Summary      Get commit history
// @Description  Get commit history for a repository
// @Tags         repositories
// @Produce      json
// @Param        id    path     string  true  "Repository ID"
// @Success      200   {array}  models.Commit
// @Failure      404   {string} string "Repository not found"
// @Security     BearerAuth
// @Router       /api/repos/{id}/commits [get]
func (h *RepositoryHandler) GetCommitHistory(w http.ResponseWriter, r *http.Request) {
	repoID := chi.URLParam(r, "id")

	h.mu.RLock()
	repo, exists := h.repositories[repoID]
	h.mu.RUnlock()

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

	release, ok := h.enterExpensiveOrReject(w, r)
	if !ok {
		return
	}
	defer release()

	commits, err := h.gitService.GetCommitHistory(repo.Path, limit)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to get commit history: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(commits)
}

// @Summary      List branches
// @Description  List all branches for a repository
// @Tags         repositories
// @Produce      json
// @Param        id    path     string  true  "Repository ID"
// @Success      200   {array}  models.Branch
// @Security     BearerAuth
// @Router       /api/repos/{id}/branches [get]
func (h *RepositoryHandler) GetBranches(w http.ResponseWriter, r *http.Request) {
	repoID := chi.URLParam(r, "id")

	h.mu.RLock()
	repo, exists := h.repositories[repoID]
	h.mu.RUnlock()

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

// @Summary      Create a commit
// @Description  Create a new commit with staged changes
// @Tags         repositories
// @Accept       json
// @Param        id    path     string              true  "Repository ID"
// @Param        body  body     models.CommitRequest  true  "Request body"
// @Success      201   {object} models.Commit
// @Failure      400   {string} string "Bad request"
// @Security     BearerAuth
// @Router       /api/repos/{id}/commit [post]
func (h *RepositoryHandler) CreateCommit(w http.ResponseWriter, r *http.Request) {
	repoID := chi.URLParam(r, "id")

	h.mu.RLock()
	repo, exists := h.repositories[repoID]
	h.mu.RUnlock()

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

// @Summary      Create a branch
// @Description  Create a new branch
// @Tags         repositories
// @Accept       json
// @Param        id    path     string  true  "Repository ID"
// @Param        name  query    string  true  "Branch name"
// @Param        from  query    string  false "Source branch (optional)"
// @Success      201   {string} string  "Created"
// @Failure      400   {string} string  "Bad request"
// @Security     BearerAuth
// @Router       /api/repos/{id}/branches [post]
func (h *RepositoryHandler) CreateBranch(w http.ResponseWriter, r *http.Request) {
	repoID := chi.URLParam(r, "id")

	h.mu.RLock()
	repo, exists := h.repositories[repoID]
	h.mu.RUnlock()

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

// @Summary      Switch branch
// @Description  Checkout a different branch
// @Tags         repositories
// @Param        id     path     string  true  "Repository ID"
// @Param        branch path     string  true  "Branch name"
// @Success      200    {string} string  "Success"
// @Security     BearerAuth
// @Router       /api/repos/{id}/branches/{branch} [put]
func (h *RepositoryHandler) SwitchBranch(w http.ResponseWriter, r *http.Request) {
	repoID := chi.URLParam(r, "id")
	branchName := chi.URLParam(r, "branch")

	h.mu.RLock()
	repo, exists := h.repositories[repoID]
	h.mu.RUnlock()

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

// @Summary      Get file tree
// @Description  Get the file tree for a repository at a specific path
// @Tags         repositories
// @Produce      json
// @Param        id    path     string  true  "Repository ID"
// @Param        path  query    string  false "Directory path (optional)"
// @Success      200   {object} models.RepoDirectoryListing
// @Security     BearerAuth
// @Router       /api/repos/{id}/files [get]
func (h *RepositoryHandler) GetFileTree(w http.ResponseWriter, r *http.Request) {
	repoID := chi.URLParam(r, "id")

	h.mu.RLock()
	repo, exists := h.repositories[repoID]
	h.mu.RUnlock()

	if !exists {
		http.Error(w, "Repository not found", http.StatusNotFound)
		return
	}

	// Parse query parameters
	path := r.URL.Query().Get("path")
	offset := parseQueryInt(r, "offset", 0)
	limit := parseQueryInt(r, "limit", 500)

	listing, err := h.gitService.BrowseDirectory(repo.Path, path, offset, limit)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to browse directory: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(listing)
}

func (h *RepositoryHandler) GetFileContent(w http.ResponseWriter, r *http.Request) {
	repoID := chi.URLParam(r, "id")
	filePath := chi.URLParam(r, "*")
	decodedPath, err := url.PathUnescape(filePath)
	if err != nil {
		decodedPath = filePath // fallback to original if decoding fails
	}

	h.mu.RLock()
	repo, exists := h.repositories[repoID]
	h.mu.RUnlock()

	if !exists {
		http.Error(w, "Repository not found", http.StatusNotFound)
		return
	}

	content, err := h.gitService.GetFileContent(repo.Path, decodedPath)
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
	decodedPath, err := url.PathUnescape(filePath)
	if err != nil {
		decodedPath = filePath // fallback to original if decoding fails
	}

	h.mu.RLock()
	repo, exists := h.repositories[repoID]
	h.mu.RUnlock()

	if !exists {
		http.Error(w, "Repository not found", http.StatusNotFound)
		return
	}

	content, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "Failed to read request body", http.StatusBadRequest)
		return
	}

	err = h.gitService.SaveFileContent(repo.Path, decodedPath, content)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to save file: %v", err), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`{"message": "File saved successfully"}`))
}

func (h *RepositoryHandler) Push(w http.ResponseWriter, r *http.Request) {
	repoID := chi.URLParam(r, "id")

	h.mu.RLock()
	repo, exists := h.repositories[repoID]
	h.mu.RUnlock()

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

	h.mu.RLock()
	repo, exists := h.repositories[repoID]
	h.mu.RUnlock()

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

	// Generate a unique ID for the repository (with mutex protection)
	h.mu.Lock()
	repoID := repoName
	counter := 1
	for _, exists := h.repositories[repoID]; exists; _, exists = h.repositories[repoID] {
		repoID = fmt.Sprintf("%s-%d", repoName, counter)
		counter++
	}
	h.mu.Unlock()

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

	// Persist to registry first (if registry exists)
	if h.registry != nil {
		regEntry := registry.Entry{
			ID:          repoID,
			Name:        repoName,
			Path:        req.Path,
			URL:         "",
			Description: "",
			Source:      "imported",
			ImportedAt:  time.Now(),
		}
		if err := h.registry.Add(regEntry); err != nil {
			http.Error(w, fmt.Sprintf("Failed to persist repository: %v", err), http.StatusInternalServerError)
			return
		}
	}

	h.mu.Lock()
	h.repositories[repoID] = repo
	h.mu.Unlock()

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(repo)
}

func (h *RepositoryHandler) Pull(w http.ResponseWriter, r *http.Request) {
	repoID := chi.URLParam(r, "id")

	h.mu.RLock()
	repo, exists := h.repositories[repoID]
	h.mu.RUnlock()

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
	decodedPath, err := url.PathUnescape(filePath)
	if err != nil {
		decodedPath = filePath // fallback to original if decoding fails
	}

	h.mu.RLock()
	repo, exists := h.repositories[repoID]
	h.mu.RUnlock()

	if !exists {
		http.Error(w, "Repository not found", http.StatusNotFound)
		return
	}

	err = h.gitService.StageFile(repo.Path, decodedPath)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to stage file: %v", err), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`{"message": "File staged successfully"}`))
}

func (h *RepositoryHandler) StageAllFiles(w http.ResponseWriter, r *http.Request) {
	repoID := chi.URLParam(r, "id")

	h.mu.RLock()
	repo, exists := h.repositories[repoID]
	h.mu.RUnlock()

	if !exists {
		http.Error(w, "Repository not found", http.StatusNotFound)
		return
	}

	err := h.gitService.StageAll(repo.Path)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to stage all files: %v", err), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`{"message": "All files staged successfully"}`))
}

func (h *RepositoryHandler) UnstageFile(w http.ResponseWriter, r *http.Request) {
	repoID := chi.URLParam(r, "id")
	filePath := chi.URLParam(r, "*")
	decodedPath, err := url.PathUnescape(filePath)
	if err != nil {
		decodedPath = filePath // fallback to original if decoding fails
	}

	h.mu.RLock()
	repo, exists := h.repositories[repoID]
	h.mu.RUnlock()

	if !exists {
		http.Error(w, "Repository not found", http.StatusNotFound)
		return
	}

	err = h.gitService.UnstageFile(repo.Path, decodedPath)
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

	h.mu.RLock()
	repo, exists := h.repositories[repoID]
	h.mu.RUnlock()

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

	h.mu.RLock()
	repo, exists := h.repositories[repoID]
	h.mu.RUnlock()

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
	decodedPath, err := url.PathUnescape(filePath)
	if err != nil {
		decodedPath = filePath // fallback to original if decoding fails
	}

	h.mu.RLock()
	repo, exists := h.repositories[repoID]
	h.mu.RUnlock()

	if !exists {
		http.Error(w, "Repository not found", http.StatusNotFound)
		return
	}

	if decodedPath == "" {
		http.Error(w, "File path is required", http.StatusBadRequest)
		return
	}

	release, ok := h.enterExpensiveOrReject(w, r)
	if !ok {
		return
	}
	defer release()

	diff, err := h.gitService.GetFileDiff(repo.Path, decodedPath)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to get file diff: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/plain")
	w.Write([]byte(diff))
}

func (h *RepositoryHandler) GenerateCommitMessage(w http.ResponseWriter, r *http.Request) {
	repoID := chi.URLParam(r, "id")

	h.mu.RLock()
	repo, exists := h.repositories[repoID]
	h.mu.RUnlock()

	if !exists {
		// Try to load the repository directly from the data path
		repoPath := filepath.Join(h.dataPath, repoID)
		if !h.isGitRepository(repoPath) {
			http.Error(w, "Repository not found", http.StatusNotFound)
			return
		}
		// Create a temporary repository object for this request
		repo = &models.Repository{
			ID:        repoID,
			Name:      repoID,
			Path:      repoPath,
			IsLocal:   true,
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		}
	}

	// Get repository status to find staged files
	status, err := h.gitService.GetRepositoryStatus(repo.Path)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to get repository status: %v", err), http.StatusInternalServerError)
		return
	}

	if len(status.Staged) == 0 {
		http.Error(w, "No staged files to generate commit message for", http.StatusBadRequest)
		return
	}

	// Generate commit message using Claude CLI
	customPrompt := ""
	if h.config != nil && h.config.ClaudePrompt != nil {
		customPrompt = *h.config.ClaudePrompt
	}

	message, err := h.claudeService.GenerateCommitMessage(nil, customPrompt)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to generate commit message: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(models.GenerateCommitMessageResponse{
		Message: message,
	})
}

func (h *RepositoryHandler) repositoryByID(repoID string) (*models.Repository, bool) {
	h.mu.RLock()
	repo, exists := h.repositories[repoID]
	h.mu.RUnlock()
	return repo, exists
}

func (h *RepositoryHandler) settingsFilePath(repoID string) string {
	return filepath.Join(h.dataPath, "settings", repoID+".json")
}

func (h *RepositoryHandler) loadRepoAppSettings(repoID string) (repoAppSettings, error) {
	settings := defaultRepoAppSettings()
	path := h.settingsFilePath(repoID)

	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return settings, nil
		}
		return repoAppSettings{}, fmt.Errorf("read settings file: %w", err)
	}

	if err := json.Unmarshal(data, &settings); err != nil {
		return repoAppSettings{}, fmt.Errorf("decode settings file: %w", err)
	}

	if settings.Sync.FetchIntervalMinutes == 0 {
		settings.Sync.FetchIntervalMinutes = 15
	}
	if settings.Sync.PullStrategy == "" {
		settings.Sync.PullStrategy = "merge"
	}
	if settings.Commit.DefaultBranch == "" {
		settings.Commit.DefaultBranch = "main"
	}
	if settings.Commit.LineEndings == "" {
		settings.Commit.LineEndings = "lf"
	}

	return settings, nil
}

func (h *RepositoryHandler) saveRepoAppSettings(repoID string, settings repoAppSettings) error {
	path := h.settingsFilePath(repoID)
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return fmt.Errorf("create settings directory: %w", err)
	}

	tmpFile, err := os.CreateTemp(filepath.Dir(path), repoID+".*.tmp")
	if err != nil {
		return fmt.Errorf("create temp settings file: %w", err)
	}
	tmpName := tmpFile.Name()
	defer os.Remove(tmpName)

	encoder := json.NewEncoder(tmpFile)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(settings); err != nil {
		return fmt.Errorf("encode settings file: %w", err)
	}

	if err := tmpFile.Sync(); err != nil {
		return fmt.Errorf("sync settings file: %w", err)
	}

	if err := tmpFile.Close(); err != nil {
		return fmt.Errorf("close temp settings file: %w", err)
	}

	if err := os.Rename(tmpName, path); err != nil {
		return fmt.Errorf("replace settings file: %w", err)
	}

	// Ensure the directory entry update is durable.
	dir, err := os.Open(filepath.Dir(path))
	if err != nil {
		return fmt.Errorf("open settings directory for sync: %w", err)
	}
	defer dir.Close()
	if err := dir.Sync(); err != nil {
		return fmt.Errorf("sync settings directory: %w", err)
	}

	return nil
}

func decodeStrictJSON(r io.Reader, dst any) error {
	dec := json.NewDecoder(r)
	dec.DisallowUnknownFields()

	if err := dec.Decode(dst); err != nil {
		return err
	}

	if dec.More() {
		return fmt.Errorf("request body must contain a single JSON object")
	}

	var extra any
	if err := dec.Decode(&extra); err != io.EOF {
		return fmt.Errorf("request body must contain a single JSON object")
	}

	return nil
}

func validateIdentitySettings(identity models.RepoIdentitySettings) error {
	identity.Name = strings.TrimSpace(identity.Name)
	identity.Email = strings.TrimSpace(identity.Email)

	if identity.Name == "" {
		return fmt.Errorf("identity name is required")
	}
	if !isBasicEmail(identity.Email) {
		return fmt.Errorf("identity email is invalid")
	}
	return nil
}

func validateSyncSettings(sync models.RepoSyncSettings) error {
	if sync.FetchIntervalMinutes != 5 && sync.FetchIntervalMinutes != 15 && sync.FetchIntervalMinutes != 30 && sync.FetchIntervalMinutes != 60 {
		return fmt.Errorf("sync.fetchIntervalMinutes must be one of 5, 15, 30, or 60")
	}
	switch sync.PullStrategy {
	case "merge", "rebase", "fast-forward":
	default:
		return fmt.Errorf("sync.pullStrategy must be one of merge, rebase, or fast-forward")
	}
	return nil
}

func validateCommitSettings(commit models.RepoCommitSettings) error {
	commit.DefaultBranch = strings.TrimSpace(commit.DefaultBranch)
	if commit.DefaultBranch == "" {
		return fmt.Errorf("commit.defaultBranch is required")
	}
	switch commit.LineEndings {
	case "lf", "crlf", "auto":
	default:
		return fmt.Errorf("commit.lineEndings must be one of lf, crlf, or auto")
	}
	return nil
}

func isBasicEmail(email string) bool {
	at := strings.Index(email, "@")
	if at <= 0 || at == len(email)-1 {
		return false
	}

	domain := email[at+1:]
	if strings.HasPrefix(domain, ".") || !strings.Contains(domain, ".") {
		return false
	}

	return true
}

func writeNoContent(w http.ResponseWriter) {
	w.WriteHeader(http.StatusNoContent)
}

func (h *RepositoryHandler) GetRepositorySettings(w http.ResponseWriter, r *http.Request) {
	repoID := chi.URLParam(r, "id")

	repo, exists := h.repositoryByID(repoID)
	if !exists {
		http.Error(w, "Repository not found", http.StatusNotFound)
		return
	}

	identity, err := h.gitService.GetGitConfig(repo.Path)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to get git config: %v", err), http.StatusInternalServerError)
		return
	}

	settings, err := h.loadRepoAppSettings(repoID)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to load repository settings: %v", err), http.StatusInternalServerError)
		return
	}

	remotes, err := h.gitService.GetRemotes(repo.Path)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to get remotes: %v", err), http.StatusInternalServerError)
		return
	}

	response := models.RepoSettings{
		Identity: models.RepoIdentitySettings{
			Name:  identity.Name,
			Email: identity.Email,
		},
		Sync:    settings.Sync,
		Commit:  settings.Commit,
		Remotes: remotes,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func (h *RepositoryHandler) UpdateRepositorySettingsIdentity(w http.ResponseWriter, r *http.Request) {
	repoID := chi.URLParam(r, "id")

	repo, exists := h.repositoryByID(repoID)
	if !exists {
		http.Error(w, "Repository not found", http.StatusNotFound)
		return
	}

	var req models.RepoIdentitySettings
	if err := decodeStrictJSON(r.Body, &req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	req.Name = strings.TrimSpace(req.Name)
	req.Email = strings.TrimSpace(req.Email)
	if err := validateIdentitySettings(req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if err := h.gitService.SetGitConfigIdentity(repo.Path, req.Name, req.Email); err != nil {
		http.Error(w, fmt.Sprintf("Failed to update git config: %v", err), http.StatusInternalServerError)
		return
	}

	writeNoContent(w)
}

func (h *RepositoryHandler) UpdateRepositorySettingsSync(w http.ResponseWriter, r *http.Request) {
	repoID := chi.URLParam(r, "id")

	_, exists := h.repositoryByID(repoID)
	if !exists {
		http.Error(w, "Repository not found", http.StatusNotFound)
		return
	}

	var req models.RepoSyncSettings
	if err := decodeStrictJSON(r.Body, &req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if err := validateSyncSettings(req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	h.settingsMu.Lock()
	defer h.settingsMu.Unlock()

	settings, err := h.loadRepoAppSettings(repoID)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to load repository settings: %v", err), http.StatusInternalServerError)
		return
	}
	settings.Sync = req

	if err := h.saveRepoAppSettings(repoID, settings); err != nil {
		http.Error(w, fmt.Sprintf("Failed to save repository settings: %v", err), http.StatusInternalServerError)
		return
	}

	writeNoContent(w)
}

func (h *RepositoryHandler) UpdateRepositorySettingsCommit(w http.ResponseWriter, r *http.Request) {
	repoID := chi.URLParam(r, "id")

	_, exists := h.repositoryByID(repoID)
	if !exists {
		http.Error(w, "Repository not found", http.StatusNotFound)
		return
	}

	var req models.RepoCommitSettings
	if err := decodeStrictJSON(r.Body, &req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	req.DefaultBranch = strings.TrimSpace(req.DefaultBranch)
	if err := validateCommitSettings(req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	h.settingsMu.Lock()
	defer h.settingsMu.Unlock()

	settings, err := h.loadRepoAppSettings(repoID)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to load repository settings: %v", err), http.StatusInternalServerError)
		return
	}
	settings.Commit = req

	if err := h.saveRepoAppSettings(repoID, settings); err != nil {
		http.Error(w, fmt.Sprintf("Failed to save repository settings: %v", err), http.StatusInternalServerError)
		return
	}

	writeNoContent(w)
}

func (h *RepositoryHandler) GetGitConfig(w http.ResponseWriter, r *http.Request) {
	repoID := chi.URLParam(r, "id")

	repo, exists := h.repositoryByID(repoID)

	if !exists {
		http.Error(w, "Repository not found", http.StatusNotFound)
		return
	}

	config, err := h.gitService.GetGitConfig(repo.Path)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to get git config: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(config)
}

// HandleTokenizedFileDiff returns a tokenized (syntax-highlighted) diff for a single file
// GET /api/repos/{id}/diff/tokenized/*?staged=<bool>&cursor=<int>&limit=<int>
func (h *RepositoryHandler) HandleTokenizedFileDiff(w http.ResponseWriter, r *http.Request) {
	repoID := chi.URLParam(r, "id")

	h.mu.RLock()
	repo, exists := h.repositories[repoID]
	h.mu.RUnlock()

	if !exists {
		http.Error(w, "Repository not found", http.StatusNotFound)
		return
	}

	// Get file path from URL wildcard
	filePath := chi.URLParam(r, "*")
	decodedPath, err := url.PathUnescape(filePath)
	if err != nil {
		decodedPath = filePath // fallback to original if decoding fails
	}

	if decodedPath == "" {
		http.Error(w, "File path is required", http.StatusBadRequest)
		return
	}

	staged := r.URL.Query().Get("staged") == "true"

	// Parse pagination args
	cursor := parseQueryInt(r, "cursor", 0)
	limit := parseQueryInt(r, "limit", 50)

	release, ok := h.enterExpensiveOrReject(w, r)
	if !ok {
		return
	}
	defer release()

	tokenizedDiff, err := h.gitService.TokenizeDiffFromPatch(repo.Path, decodedPath, staged, cursor, limit)
	if err != nil {
		http.Error(w, "Failed to get tokenized diff: "+err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(tokenizedDiff)
}

// HandleTokenizedCommitDiff returns tokenized diffs for all files in a commit
// GET /api/repos/{id}/diff/commit/tokenized?hash=<commit>
func (h *RepositoryHandler) HandleTokenizedCommitDiff(w http.ResponseWriter, r *http.Request) {
	repoID := chi.URLParam(r, "id")

	h.mu.RLock()
	repo, exists := h.repositories[repoID]
	h.mu.RUnlock()

	if !exists {
		http.Error(w, "Repository not found", http.StatusNotFound)
		return
	}

	hash := r.URL.Query().Get("hash")
	if hash == "" {
		http.Error(w, "Commit hash is required", http.StatusBadRequest)
		return
	}

	release, ok := h.enterExpensiveOrReject(w, r)
	if !ok {
		return
	}
	defer release()

	tokenizedDiff, err := h.gitService.TokenizeCommitDiff(repo.Path, hash)
	if err != nil {
		status := http.StatusInternalServerError
		msg := "Failed to get tokenized commit diff: " + err.Error()
		if strings.Contains(err.Error(), "commit not found") || strings.Contains(err.Error(), "failed to get commit") {
			status = http.StatusNotFound
			msg = "Commit not found"
		}
		http.Error(w, msg, status)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(tokenizedDiff)
}

// HandleCommitFileDiff returns a tokenized diff for a specific file at a specific commit
// GET /api/repos/{id}/diff/commit/{hash}/files/{path}?cursor=<int>&limit=<int>
func (h *RepositoryHandler) HandleCommitFileDiff(w http.ResponseWriter, r *http.Request) {
	repoID := chi.URLParam(r, "id")
	commitHash := chi.URLParam(r, "hash")

	h.mu.RLock()
	repo, exists := h.repositories[repoID]
	h.mu.RUnlock()

	if !exists {
		http.Error(w, "Repository not found", http.StatusNotFound)
		return
	}

	// Get file path from URL wildcard
	filePath := chi.URLParam(r, "*")
	decodedPath, err := url.PathUnescape(filePath)
	if err != nil {
		decodedPath = filePath // fallback to original if decoding fails
	}

	if decodedPath == "" {
		http.Error(w, "File path is required", http.StatusBadRequest)
		return
	}

	if commitHash == "" {
		http.Error(w, "Commit hash is required", http.StatusBadRequest)
		return
	}

	// Parse pagination args
	cursor := parseQueryInt(r, "cursor", 0)
	limit := parseQueryInt(r, "limit", 50)

	release, ok := h.enterExpensiveOrReject(w, r)
	if !ok {
		return
	}
	defer release()

	tokenizedDiff, err := h.gitService.GetCommitFileDiff(repo.Path, commitHash, decodedPath, cursor, limit)
	if err != nil {
		// Determine appropriate status code
		status := http.StatusInternalServerError
		msg := "Failed to get commit file diff: " + err.Error()

		if strings.Contains(err.Error(), "commit not found") {
			status = http.StatusNotFound
			msg = "Commit not found"
		} else if strings.Contains(err.Error(), "file not found") {
			status = http.StatusNotFound
			msg = "File not found in this commit"
		}

		http.Error(w, msg, status)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(tokenizedDiff)
}

// parseQueryInt parses an integer from query string with a default value
func parseQueryInt(r *http.Request, key string, defaultValue int) int {
	str := r.URL.Query().Get(key)
	if str == "" {
		return defaultValue
	}
	val, err := strconv.Atoi(str)
	if err != nil {
		return defaultValue
	}
	return val
}
