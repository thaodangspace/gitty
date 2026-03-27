package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	gogitconfig "github.com/go-git/go-git/v5/config"
	"gitweb/server/internal/git"
	"gitweb/server/internal/models"
	"gitweb/server/internal/registry"
)

// Helper function to create a properly initialized test repository
func createTestRepository(handler *RepositoryHandler, repoName string) (string, error) {
	tempDir, err := os.MkdirTemp("", "handler_test")
	if err != nil {
		return "", err
	}

	repoDir := filepath.Join(tempDir, repoName)
	err = os.Mkdir(repoDir, 0755)
	if err != nil {
		return "", err
	}

	// Initialize git repository properly
	_, err = handler.gitService.InitRepository(repoDir)
	if err != nil {
		return "", err
	}

	// Create initial commit
	testFile := filepath.Join(repoDir, "README.md")
	err = os.WriteFile(testFile, []byte("# Test Repository"), 0644)
	if err != nil {
		return "", err
	}

	err = handler.gitService.StageFile(repoDir, "README.md")
	if err != nil {
		return "", err
	}

	commitReq := models.CommitRequest{
		Message: "Initial commit",
		Files:   []string{"README.md"},
		Author: models.Author{
			Name:  "Test User",
			Email: "test@example.com",
		},
	}

	err = handler.gitService.CreateCommit(repoDir, commitReq)
	if err != nil {
		return "", err
	}

	// Add repository to handler
	handler.repositories[repoName] = &models.Repository{
		ID:        repoName,
		Name:      repoName,
		Path:      repoDir,
		IsLocal:   true,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	return repoDir, nil
}

func newRepoSettingsRequest(method, targetURL, repoID string, body []byte) *http.Request {
	req := httptest.NewRequest(method, targetURL, bytes.NewBuffer(body))
	ctx := chi.NewRouteContext()
	ctx.URLParams.Add("id", repoID)
	return req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, ctx))
}

func TestNewRepositoryHandler(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "handler_test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tempDir)

	handler := NewRepositoryHandler(tempDir, nil, nil)
	if handler == nil {
		t.Fatal("NewRepositoryHandler returned nil")
	}

	if handler.dataPath != tempDir {
		t.Errorf("Expected dataPath %s, got %s", tempDir, handler.dataPath)
	}

	if handler.gitService == nil {
		t.Error("Git service should be initialized")
	}

	if handler.repositories == nil {
		t.Error("Repositories map should be initialized")
	}
}

func TestIsGitRepository(t *testing.T) {
	handler := NewRepositoryHandler("", nil, nil)

	tempDir, err := os.MkdirTemp("", "handler_test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tempDir)

	// Test non-git directory
	if handler.isGitRepository(tempDir) {
		t.Error("Empty directory should not be a git repository")
	}

	// Create .git directory
	gitDir := filepath.Join(tempDir, ".git")
	err = os.Mkdir(gitDir, 0755)
	if err != nil {
		t.Fatal(err)
	}

	// Test git directory
	if !handler.isGitRepository(tempDir) {
		t.Error("Directory with .git should be a git repository")
	}
}

func TestListRepositories(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "handler_test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tempDir)

	handler := NewRepositoryHandler(tempDir, nil, nil)

	// Create a test repository
	repoDir := filepath.Join(tempDir, "test-repo")
	err = os.Mkdir(repoDir, 0755)
	if err != nil {
		t.Fatal(err)
	}

	// Initialize git repository
	gitDir := filepath.Join(repoDir, ".git")
	err = os.Mkdir(gitDir, 0755)
	if err != nil {
		t.Fatal(err)
	}

	// Manually add repository to handler (since we no longer scan dataPath)
	handler.mu.Lock()
	handler.repositories["test-repo"] = &models.Repository{
		ID:        "test-repo",
		Name:      "test-repo",
		Path:      repoDir,
		IsLocal:   true,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	handler.mu.Unlock()

	// Create HTTP request
	req := httptest.NewRequest("GET", "/repositories", nil)
	w := httptest.NewRecorder()

	// Call handler
	handler.ListRepositories(w, req)

	// Check response
	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	// Parse response
	var repos []*models.Repository
	err = json.Unmarshal(w.Body.Bytes(), &repos)
	if err != nil {
		t.Fatal(err)
	}

	if len(repos) != 1 {
		t.Errorf("Expected 1 repository, got %d", len(repos))
	}

	repo := repos[0]
	if repo.Name != "test-repo" {
		t.Errorf("Expected repository name 'test-repo', got '%s'", repo.Name)
	}

	if repo.Path != repoDir {
		t.Errorf("Expected repository path %s, got %s", repoDir, repo.Path)
	}

	if !repo.IsLocal {
		t.Error("Repository should be marked as local")
	}
}

func TestCreateRepository(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "handler_test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tempDir)

	handler := NewRepositoryHandler(tempDir, nil, nil)

	// Test creating a repository
	createReq := models.CreateRepositoryRequest{
		Name:        "new-repo",
		Description: "Test repository",
		IsLocal:     true,
	}

	reqBody, err := json.Marshal(createReq)
	if err != nil {
		t.Fatal(err)
	}

	req := httptest.NewRequest("POST", "/repositories", bytes.NewBuffer(reqBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	// Call handler
	handler.CreateRepository(w, req)

	// Check response
	if w.Code != http.StatusCreated {
		t.Errorf("Expected status 201, got %d", w.Code)
	}

	// Verify repository was created
	repoPath := filepath.Join(tempDir, "new-repo")
	if _, err := os.Stat(repoPath); os.IsNotExist(err) {
		t.Error("Repository directory was not created")
	}

	// Check that .git directory exists
	gitPath := filepath.Join(repoPath, ".git")
	if _, err := os.Stat(gitPath); os.IsNotExist(err) {
		t.Error(".git directory was not created")
	}
}

func TestCreateRepositoryInvalidRequest(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "handler_test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tempDir)

	handler := NewRepositoryHandler(tempDir, nil, nil)

	// Test with empty name
	createReq := models.CreateRepositoryRequest{
		Name: "",
	}

	reqBody, err := json.Marshal(createReq)
	if err != nil {
		t.Fatal(err)
	}

	req := httptest.NewRequest("POST", "/repositories", bytes.NewBuffer(reqBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	// Call handler
	handler.CreateRepository(w, req)

	// Check response
	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400, got %d", w.Code)
	}
}

func TestGetRepository(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "handler_test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tempDir)

	handler := NewRepositoryHandler(tempDir, nil, nil)

	// Create a test repository
	repoDir := filepath.Join(tempDir, "test-repo")
	err = os.Mkdir(repoDir, 0755)
	if err != nil {
		t.Fatal(err)
	}

	gitDir := filepath.Join(repoDir, ".git")
	err = os.Mkdir(gitDir, 0755)
	if err != nil {
		t.Fatal(err)
	}

	// Add repository to handler
	handler.repositories["test-repo"] = &models.Repository{
		ID:        "test-repo",
		Name:      "test-repo",
		Path:      repoDir,
		IsLocal:   true,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	// Create HTTP request with repository ID
	req := httptest.NewRequest("GET", "/repositories/test-repo", nil)
	w := httptest.NewRecorder()

	// Set up chi router context
	chiCtx := chi.NewRouteContext()
	chiCtx.URLParams.Add("id", "test-repo")
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, chiCtx))

	// Call handler
	handler.GetRepository(w, req)

	// Check response
	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	// Parse response
	var repo models.Repository
	err = json.Unmarshal(w.Body.Bytes(), &repo)
	if err != nil {
		t.Fatal(err)
	}

	if repo.ID != "test-repo" {
		t.Errorf("Expected repository ID 'test-repo', got '%s'", repo.ID)
	}
}

func TestGetRepositoryNotFound(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "handler_test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tempDir)

	handler := NewRepositoryHandler(tempDir, nil, nil)

	// Create HTTP request with non-existent repository ID
	req := httptest.NewRequest("GET", "/repositories/non-existent", nil)
	w := httptest.NewRecorder()

	// Set up chi router context
	chiCtx := chi.NewRouteContext()
	chiCtx.URLParams.Add("id", "non-existent")
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, chiCtx))

	// Call handler
	handler.GetRepository(w, req)

	// Check response
	if w.Code != http.StatusNotFound {
		t.Errorf("Expected status 404, got %d", w.Code)
	}
}

func TestDeleteRepository(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "handler_test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tempDir)

	handler := NewRepositoryHandler(tempDir, nil, nil)

	// Create a test repository
	repoDir := filepath.Join(tempDir, "test-repo")
	err = os.Mkdir(repoDir, 0755)
	if err != nil {
		t.Fatal(err)
	}

	gitDir := filepath.Join(repoDir, ".git")
	err = os.Mkdir(gitDir, 0755)
	if err != nil {
		t.Fatal(err)
	}

	// Add repository to handler
	handler.mu.Lock()
	handler.repositories["test-repo"] = &models.Repository{
		ID:        "test-repo",
		Name:      "test-repo",
		Path:      repoDir,
		IsLocal:   true,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	handler.mu.Unlock()

	// Create HTTP request
	req := httptest.NewRequest("DELETE", "/repositories/test-repo", nil)
	w := httptest.NewRecorder()

	// Set up chi router context
	chiCtx := chi.NewRouteContext()
	chiCtx.URLParams.Add("id", "test-repo")
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, chiCtx))

	// Call handler
	handler.DeleteRepository(w, req)

	// Check response
	if w.Code != http.StatusNoContent {
		t.Errorf("Expected status 204, got %d", w.Code)
	}

	// Verify repository was removed from handler (but directory is NOT deleted)
	handler.mu.RLock()
	_, exists := handler.repositories["test-repo"]
	handler.mu.RUnlock()
	if exists {
		t.Error("Repository should be removed from handler")
	}

	// Repository directory should still exist (we don't delete files anymore)
	if _, err := os.Stat(repoDir); os.IsNotExist(err) {
		t.Error("Repository directory should not be deleted")
	}
}

func TestGetRepositoryStatus(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "handler_test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tempDir)

	handler := NewRepositoryHandler(tempDir, nil, nil)

	// Create a properly initialized test repository
	_, err = createTestRepository(handler, "test-repo")
	if err != nil {
		t.Fatal(err)
	}

	// Create HTTP request
	req := httptest.NewRequest("GET", "/repositories/test-repo/status", nil)
	w := httptest.NewRecorder()

	// Set up chi router context
	chiCtx := chi.NewRouteContext()
	chiCtx.URLParams.Add("id", "test-repo")
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, chiCtx))

	// Call handler
	handler.GetRepositoryStatus(w, req)

	// Check response
	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	// Parse response
	var status models.RepositoryStatus
	err = json.Unmarshal(w.Body.Bytes(), &status)
	if err != nil {
		t.Fatal(err)
	}

	if status.RepositoryID != "test-repo" {
		t.Errorf("Expected repository ID 'test-repo', got '%s'", status.RepositoryID)
	}
}

func TestGetRepositoryStatus_UntrackedNotInStaged(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "handler_test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tempDir)

	handler := NewRepositoryHandler(tempDir, nil, nil)
	repoDir, err := createTestRepository(handler, "test-repo")
	if err != nil {
		t.Fatal(err)
	}

	if err := os.WriteFile(filepath.Join(repoDir, "new.txt"), []byte("new"), 0644); err != nil {
		t.Fatal(err)
	}

	req := httptest.NewRequest("GET", "/repositories/test-repo/status", nil)
	w := httptest.NewRecorder()

	chiCtx := chi.NewRouteContext()
	chiCtx.URLParams.Add("id", "test-repo")
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, chiCtx))

	handler.GetRepositoryStatus(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("Expected status 200, got %d", w.Code)
	}

	var status models.RepositoryStatus
	if err := json.Unmarshal(w.Body.Bytes(), &status); err != nil {
		t.Fatal(err)
	}

	for _, staged := range status.Staged {
		if staged.Status == "?" {
			t.Fatalf("staged must not contain '?': %+v", staged)
		}
		if staged.Path == "new.txt" {
			t.Fatalf("untracked file must not appear in staged: %+v", staged)
		}
	}
}

func TestGetCommitHistory(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "handler_test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tempDir)

	handler := NewRepositoryHandler(tempDir, nil, nil)

	// Create a properly initialized test repository
	_, err = createTestRepository(handler, "test-repo")
	if err != nil {
		t.Fatal(err)
	}

	// Create HTTP request
	req := httptest.NewRequest("GET", "/repositories/test-repo/commits", nil)
	w := httptest.NewRecorder()

	// Set up chi router context
	chiCtx := chi.NewRouteContext()
	chiCtx.URLParams.Add("id", "test-repo")
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, chiCtx))

	// Call handler
	handler.GetCommitHistory(w, req)

	// Check response
	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	// Parse response
	var commits []models.Commit
	err = json.Unmarshal(w.Body.Bytes(), &commits)
	if err != nil {
		t.Fatal(err)
	}

	// Should have at least 1 commit (initial commit)
	if len(commits) < 1 {
		t.Errorf("Expected at least 1 commit, got %d", len(commits))
	}
}

func TestGetBranches(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "handler_test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tempDir)

	handler := NewRepositoryHandler(tempDir, nil, nil)

	// Create a properly initialized test repository
	_, err = createTestRepository(handler, "test-repo")
	if err != nil {
		t.Fatal(err)
	}

	// Create HTTP request
	req := httptest.NewRequest("GET", "/repositories/test-repo/branches", nil)
	w := httptest.NewRecorder()

	// Set up chi router context
	chiCtx := chi.NewRouteContext()
	chiCtx.URLParams.Add("id", "test-repo")
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, chiCtx))

	// Call handler
	handler.GetBranches(w, req)

	// Check response
	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	// Parse response
	var branches []models.Branch
	err = json.Unmarshal(w.Body.Bytes(), &branches)
	if err != nil {
		t.Fatal(err)
	}

	// Should have at least one branch (main/master)
	if len(branches) == 0 {
		t.Error("Should have at least one branch")
	}
}

func TestCreateCommit(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "handler_test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tempDir)

	handler := NewRepositoryHandler(tempDir, nil, nil)

	// Create a properly initialized test repository
	repoDir, err := createTestRepository(handler, "test-repo")
	if err != nil {
		t.Fatal(err)
	}

	// Create a test file
	testFile := filepath.Join(repoDir, "test.txt")
	err = os.WriteFile(testFile, []byte("test content"), 0644)
	if err != nil {
		t.Fatal(err)
	}

	// Create commit request
	commitReq := models.CommitRequest{
		Message: "Test commit",
		Files:   []string{"test.txt"},
		Author: models.Author{
			Name:  "Test User",
			Email: "test@example.com",
		},
	}

	reqBody, err := json.Marshal(commitReq)
	if err != nil {
		t.Fatal(err)
	}

	// Create HTTP request
	req := httptest.NewRequest("POST", "/repositories/test-repo/commits", bytes.NewBuffer(reqBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	// Set up chi router context
	chiCtx := chi.NewRouteContext()
	chiCtx.URLParams.Add("id", "test-repo")
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, chiCtx))

	// Call handler
	handler.CreateCommit(w, req)

	// Check response
	if w.Code != http.StatusCreated {
		t.Errorf("Expected status 201, got %d", w.Code)
	}
}

func TestCreateBranch(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "handler_test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tempDir)

	handler := NewRepositoryHandler(tempDir, nil, nil)

	// Create a properly initialized test repository
	_, err = createTestRepository(handler, "test-repo")
	if err != nil {
		t.Fatal(err)
	}

	// Create branch request
	branchReq := struct {
		Name string `json:"name"`
	}{
		Name: "new-branch",
	}

	reqBody, err := json.Marshal(branchReq)
	if err != nil {
		t.Fatal(err)
	}

	// Create HTTP request
	req := httptest.NewRequest("POST", "/repositories/test-repo/branches", bytes.NewBuffer(reqBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	// Set up chi router context
	chiCtx := chi.NewRouteContext()
	chiCtx.URLParams.Add("id", "test-repo")
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, chiCtx))

	// Call handler
	handler.CreateBranch(w, req)

	// Check response
	if w.Code != http.StatusCreated {
		t.Errorf("Expected status 201, got %d", w.Code)
	}
}

func TestSwitchBranch(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "handler_test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tempDir)

	handler := NewRepositoryHandler(tempDir, nil, nil)

	// Create a properly initialized test repository
	_, err = createTestRepository(handler, "test-repo")
	if err != nil {
		t.Fatal(err)
	}

	// Get repository path with mutex protection
	handler.mu.RLock()
	repo := handler.repositories["test-repo"]
	handler.mu.RUnlock()

	// Create a new branch first
	err = handler.gitService.CreateBranch(repo.Path, "feature-branch")
	if err != nil {
		t.Fatal(err)
	}

	// Create HTTP request to switch to the new branch
	req := httptest.NewRequest("POST", "/repositories/test-repo/branches/switch/feature-branch", nil)
	w := httptest.NewRecorder()

	// Set up chi router context
	chiCtx := chi.NewRouteContext()
	chiCtx.URLParams.Add("id", "test-repo")
	chiCtx.URLParams.Add("branch", "feature-branch")
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, chiCtx))

	// Call handler
	handler.SwitchBranch(w, req)

	// Check response
	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d: %s", w.Code, w.Body.String())
	}
}

func TestGetFileTree(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "handler_test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tempDir)

	handler := NewRepositoryHandler(tempDir, nil, nil)

	// Create a test repository
	repoDir := filepath.Join(tempDir, "test-repo")
	err = os.Mkdir(repoDir, 0755)
	if err != nil {
		t.Fatal(err)
	}

	// Initialize git repository
	gitDir := filepath.Join(repoDir, ".git")
	err = os.Mkdir(gitDir, 0755)
	if err != nil {
		t.Fatal(err)
	}

	// Create some test files
	testFiles := []string{
		"file1.txt",
		"dir1/file2.txt",
	}

	for _, filePath := range testFiles {
		// Create directory if needed
		dir := filepath.Dir(filePath)
		if dir != "." {
			err = os.MkdirAll(filepath.Join(repoDir, dir), 0755)
			if err != nil {
				t.Fatal(err)
			}
		}

		// Create file
		err = os.WriteFile(filepath.Join(repoDir, filePath), []byte("test content"), 0644)
		if err != nil {
			t.Fatal(err)
		}
	}

	// Add repository to handler
	handler.repositories["test-repo"] = &models.Repository{
		ID:        "test-repo",
		Name:      "test-repo",
		Path:      repoDir,
		IsLocal:   true,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	// Create HTTP request
	req := httptest.NewRequest("GET", "/repositories/test-repo/files", nil)
	w := httptest.NewRecorder()

	// Set up chi router context
	chiCtx := chi.NewRouteContext()
	chiCtx.URLParams.Add("id", "test-repo")
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, chiCtx))

	// Call handler
	handler.GetFileTree(w, req)

	// Check response
	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	// Parse response
	var listing models.RepoDirectoryListing
	err = json.Unmarshal(w.Body.Bytes(), &listing)
	if err != nil {
		t.Fatal(err)
	}

	// Should have at least the test files/directories
	if len(listing.Entries) < len(testFiles) {
		t.Errorf("Expected at least %d entries, got %d", len(testFiles), len(listing.Entries))
	}
}

func TestGetFileContent(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "handler_test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tempDir)

	handler := NewRepositoryHandler(tempDir, nil, nil)

	// Create a test repository
	repoDir := filepath.Join(tempDir, "test-repo")
	err = os.Mkdir(repoDir, 0755)
	if err != nil {
		t.Fatal(err)
	}

	// Initialize git repository
	gitDir := filepath.Join(repoDir, ".git")
	err = os.Mkdir(gitDir, 0755)
	if err != nil {
		t.Fatal(err)
	}

	// Create a test file
	testContent := "Hello, World!"
	testFile := filepath.Join(repoDir, "test.txt")
	err = os.WriteFile(testFile, []byte(testContent), 0644)
	if err != nil {
		t.Fatal(err)
	}

	// Add repository to handler
	handler.repositories["test-repo"] = &models.Repository{
		ID:        "test-repo",
		Name:      "test-repo",
		Path:      repoDir,
		IsLocal:   true,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	// Create HTTP request
	req := httptest.NewRequest("GET", "/repositories/test-repo/files/test.txt", nil)
	w := httptest.NewRecorder()

	// Set up chi router context
	chiCtx := chi.NewRouteContext()
	chiCtx.URLParams.Add("id", "test-repo")
	chiCtx.URLParams.Add("*", "test.txt")
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, chiCtx))

	// Call handler
	handler.GetFileContent(w, req)

	// Check response
	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	// Check content
	if w.Body.String() != testContent {
		t.Errorf("Expected content '%s', got '%s'", testContent, w.Body.String())
	}
}

func TestSaveFileContent(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "handler_test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tempDir)

	handler := NewRepositoryHandler(tempDir, nil, nil)

	// Create a test repository
	repoDir := filepath.Join(tempDir, "test-repo")
	err = os.Mkdir(repoDir, 0755)
	if err != nil {
		t.Fatal(err)
	}

	// Initialize git repository
	gitDir := filepath.Join(repoDir, ".git")
	err = os.Mkdir(gitDir, 0755)
	if err != nil {
		t.Fatal(err)
	}

	// Add repository to handler
	handler.repositories["test-repo"] = &models.Repository{
		ID:        "test-repo",
		Name:      "test-repo",
		Path:      repoDir,
		IsLocal:   true,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	// Create save file request - just send the content directly
	reqBody := []byte("New content")

	// Create HTTP request
	req := httptest.NewRequest("PUT", "/repositories/test-repo/files/test.txt", bytes.NewBuffer(reqBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	// Set up chi router context
	chiCtx := chi.NewRouteContext()
	chiCtx.URLParams.Add("id", "test-repo")
	chiCtx.URLParams.Add("*", "test.txt")
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, chiCtx))

	// Call handler
	handler.SaveFileContent(w, req)

	// Check response
	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	// Verify file was created
	testFile := filepath.Join(repoDir, "test.txt")
	content, err := os.ReadFile(testFile)
	if err != nil {
		t.Fatal(err)
	}

	if string(content) != "New content" {
		t.Errorf("Expected content 'New content', got '%s'", string(content))
	}

	// Check response message
	var response map[string]string
	err = json.Unmarshal(w.Body.Bytes(), &response)
	if err != nil {
		t.Fatal(err)
	}

	if response["message"] != "File saved successfully" {
		t.Errorf("Expected message 'File saved successfully', got '%s'", response["message"])
	}
}

func TestStageFile(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "handler_test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tempDir)

	handler := NewRepositoryHandler(tempDir, nil, nil)

	// Create a properly initialized test repository
	repoDir, err := createTestRepository(handler, "test-repo")
	if err != nil {
		t.Fatal(err)
	}

	// Create a test file
	testFile := filepath.Join(repoDir, "test.txt")
	err = os.WriteFile(testFile, []byte("test content"), 0644)
	if err != nil {
		t.Fatal(err)
	}

	// Create HTTP request with file path in URL
	req := httptest.NewRequest("POST", "/repositories/test-repo/stage/test.txt", nil)
	w := httptest.NewRecorder()

	// Set up chi router context
	chiCtx := chi.NewRouteContext()
	chiCtx.URLParams.Add("id", "test-repo")
	chiCtx.URLParams.Add("*", "test.txt")
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, chiCtx))

	// Call handler
	handler.StageFile(w, req)

	// Check response
	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}
}

func TestUnstageFile(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "handler_test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tempDir)

	handler := NewRepositoryHandler(tempDir, nil, nil)

	// Create a properly initialized test repository
	repoDir, err := createTestRepository(handler, "test-repo")
	if err != nil {
		t.Fatal(err)
	}

	// Create a test file
	testFile := filepath.Join(repoDir, "test.txt")
	err = os.WriteFile(testFile, []byte("test content"), 0644)
	if err != nil {
		t.Fatal(err)
	}

	// Create HTTP request with file path in URL
	req := httptest.NewRequest("POST", "/repositories/test-repo/unstage/test.txt", nil)
	w := httptest.NewRecorder()

	// Set up chi router context
	chiCtx := chi.NewRouteContext()
	chiCtx.URLParams.Add("id", "test-repo")
	chiCtx.URLParams.Add("*", "test.txt")
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, chiCtx))

	// Call handler
	handler.UnstageFile(w, req)

	// Check response
	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}
}

func TestGetFileTree_QueryParams(t *testing.T) {
	// Setup handler with a test repository
	tempDir, err := os.MkdirTemp("", "git_test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tempDir)

	// Initialize git repo
	gitService := git.NewService()
	_, err = gitService.InitRepository(tempDir)
	if err != nil {
		t.Fatal(err)
	}

	// Create test structure
	os.MkdirAll(filepath.Join(tempDir, "src", "components"), 0755)
	os.WriteFile(filepath.Join(tempDir, "src", "index.ts"), []byte("test"), 0644)
	os.WriteFile(filepath.Join(tempDir, "README.md"), []byte("readme"), 0644)

	// Create handler
	handler := NewRepositoryHandler(tempDir, nil, nil)

	// Add repository to handler
	handler.repositories["test-repo"] = &models.Repository{
		ID:        "test-repo",
		Name:      "test-repo",
		Path:      tempDir,
		IsLocal:   true,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	// Use chi router for URL param extraction
	r := chi.NewRouter()
	r.Get("/repos/{id}/files", handler.GetFileTree)

	// Test root directory
	req := httptest.NewRequest("GET", "/repos/test-repo/files", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d: %s", w.Code, w.Body.String())
	}

	var listing models.RepoDirectoryListing
	json.Unmarshal(w.Body.Bytes(), &listing)

	// Root should have README.md and src/
	if len(listing.Entries) < 2 {
		t.Errorf("Expected at least 2 entries in root, got %d", len(listing.Entries))
	}

	// Test subdirectory
	req = httptest.NewRequest("GET", "/repos/test-repo/files?path=src", nil)
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d: %s", w.Code, w.Body.String())
	}

	json.Unmarshal(w.Body.Bytes(), &listing)

	if listing.Path != "src" {
		t.Errorf("Expected path 'src', got %s", listing.Path)
	}

	// Test pagination
	req = httptest.NewRequest("GET", "/repos/test-repo/files?offset=0&limit=1", nil)
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d: %s", w.Code, w.Body.String())
	}

	json.Unmarshal(w.Body.Bytes(), &listing)

	if len(listing.Entries) > 1 {
		t.Errorf("Expected at most 1 entry with limit=1, got %d", len(listing.Entries))
	}
	if !listing.HasMore || listing.TotalCount < 2 {
		t.Errorf("Expected has_more=true with total_count >= 2, got has_more=%v, total_count=%d", listing.HasMore, listing.TotalCount)
	}
}

func TestImportRepositoryPersistsToRegistry(t *testing.T) {
	tmp := t.TempDir()
	dataPath := filepath.Join(tmp, "data")
	os.MkdirAll(dataPath, 0755)

	regPath := filepath.Join(tmp, "registry", "repository.json")
	reg, err := registry.New(regPath)
	if err != nil {
		t.Fatalf("registry.New() error: %v", err)
	}

	// Create a git repo to import
	repoPath := filepath.Join(tmp, "my-repo")
	os.MkdirAll(repoPath, 0755)
	svc := git.NewService()
	svc.InitRepository(repoPath)

	handler := NewRepositoryHandler(dataPath, nil, reg)

	// Import via handler
	body := bytes.NewBufferString(`{"path":"` + repoPath + `"}`)
	req := httptest.NewRequest("POST", "/api/repos/import", body)
	w := httptest.NewRecorder()
	handler.ImportRepository(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", w.Code, w.Body.String())
	}

	// Verify registry has the entry
	entries := reg.List()
	if len(entries) != 1 {
		t.Fatalf("expected 1 registry entry, got %d", len(entries))
	}
	if entries[0].Source != "imported" {
		t.Fatalf("expected source 'imported', got %q", entries[0].Source)
	}

	// Create a new handler with the same registry — simulates server restart
	handler2 := NewRepositoryHandler(dataPath, nil, reg)
	_ = handler2 // loadRepositories runs in constructor

	// Verify the repo is loaded from registry
	req2 := httptest.NewRequest("GET", "/api/repos", nil)
	w2 := httptest.NewRecorder()
	handler2.ListRepositories(w2, req2)

	var repos []models.Repository
	json.Unmarshal(w2.Body.Bytes(), &repos)
	if len(repos) != 1 {
		t.Fatalf("expected 1 repo after reload, got %d", len(repos))
	}
	if repos[0].Path != repoPath {
		t.Fatalf("expected path %q, got %q", repoPath, repos[0].Path)
	}
}

func TestRepositorySettingsHandlers_HappyPath(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "handler_settings_test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tempDir)

	handler := NewRepositoryHandler(tempDir, nil, nil)
	repoName := "settings-repo"
	repoDir, err := createTestRepository(handler, repoName)
	if err != nil {
		t.Fatal(err)
	}

	if err := handler.gitService.SetGitConfigIdentity(repoDir, "Test User", "test@example.com"); err != nil {
		t.Fatal(err)
	}

	repo, err := handler.gitService.OpenRepository(repoDir)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := repo.CreateRemote(&gogitconfig.RemoteConfig{
		Name: "origin",
		URLs: []string{"https://example.com/test/repo.git"},
	}); err != nil {
		t.Fatal(err)
	}

	initialSettings := repoAppSettings{
		Sync: models.RepoSyncSettings{
			AutoFetch:            true,
			FetchIntervalMinutes: 30,
			PullStrategy:         "rebase",
		},
		Commit: models.RepoCommitSettings{
			DefaultBranch:  "develop",
			SigningEnabled: true,
			LineEndings:    "crlf",
		},
	}
	if err := handler.saveRepoAppSettings(repoName, initialSettings); err != nil {
		t.Fatal(err)
	}

	getReq := newRepoSettingsRequest(http.MethodGet, "/api/repos/"+repoName+"/settings", repoName, nil)
	getRec := httptest.NewRecorder()
	handler.GetRepositorySettings(getRec, getReq)

	if getRec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", getRec.Code, getRec.Body.String())
	}

	var settings models.RepoSettings
	if err := json.Unmarshal(getRec.Body.Bytes(), &settings); err != nil {
		t.Fatal(err)
	}

	if settings.Identity.Name != "Test User" || settings.Identity.Email != "test@example.com" {
		t.Fatalf("unexpected identity: %+v", settings.Identity)
	}
	if settings.Sync != initialSettings.Sync {
		t.Fatalf("unexpected sync settings: %+v", settings.Sync)
	}
	if settings.Commit != initialSettings.Commit {
		t.Fatalf("unexpected commit settings: %+v", settings.Commit)
	}
	if len(settings.Remotes) != 1 || settings.Remotes[0].Name != "origin" || settings.Remotes[0].URL != "https://example.com/test/repo.git" {
		t.Fatalf("unexpected remotes: %+v", settings.Remotes)
	}

	identityBody := []byte(`{"name":"Updated User","email":"updated@example.com"}`)
	identityReq := newRepoSettingsRequest(http.MethodPut, "/api/repos/"+repoName+"/settings/identity", repoName, identityBody)
	identityRec := httptest.NewRecorder()
	handler.UpdateRepositorySettingsIdentity(identityRec, identityReq)
	if identityRec.Code != http.StatusNoContent {
		t.Fatalf("expected 204 for identity update, got %d: %s", identityRec.Code, identityRec.Body.String())
	}

	updatedIdentity, err := handler.gitService.GetGitConfig(repoDir)
	if err != nil {
		t.Fatal(err)
	}
	if updatedIdentity.Name != "Updated User" || updatedIdentity.Email != "updated@example.com" {
		t.Fatalf("identity not updated: %+v", updatedIdentity)
	}

	syncBody := []byte(`{"autoFetch":true,"fetchIntervalMinutes":60,"pullStrategy":"fast-forward"}`)
	syncReq := newRepoSettingsRequest(http.MethodPut, "/api/repos/"+repoName+"/settings/sync", repoName, syncBody)
	syncRec := httptest.NewRecorder()
	handler.UpdateRepositorySettingsSync(syncRec, syncReq)
	if syncRec.Code != http.StatusNoContent {
		t.Fatalf("expected 204 for sync update, got %d: %s", syncRec.Code, syncRec.Body.String())
	}

	persistedSettings, err := handler.loadRepoAppSettings(repoName)
	if err != nil {
		t.Fatal(err)
	}
	if persistedSettings.Sync.AutoFetch != true || persistedSettings.Sync.FetchIntervalMinutes != 60 || persistedSettings.Sync.PullStrategy != "fast-forward" {
		t.Fatalf("sync not updated: %+v", persistedSettings.Sync)
	}
	if persistedSettings.Commit != initialSettings.Commit {
		t.Fatalf("commit settings should be unchanged after sync update: %+v", persistedSettings.Commit)
	}

	commitBody := []byte(`{"defaultBranch":"trunk","signingEnabled":true,"lineEndings":"auto"}`)
	commitReq := newRepoSettingsRequest(http.MethodPut, "/api/repos/"+repoName+"/settings/commit", repoName, commitBody)
	commitRec := httptest.NewRecorder()
	handler.UpdateRepositorySettingsCommit(commitRec, commitReq)
	if commitRec.Code != http.StatusNoContent {
		t.Fatalf("expected 204 for commit update, got %d: %s", commitRec.Code, commitRec.Body.String())
	}

	persistedSettings, err = handler.loadRepoAppSettings(repoName)
	if err != nil {
		t.Fatal(err)
	}
	if persistedSettings.Commit.DefaultBranch != "trunk" || !persistedSettings.Commit.SigningEnabled || persistedSettings.Commit.LineEndings != "auto" {
		t.Fatalf("commit not updated: %+v", persistedSettings.Commit)
	}
}

func TestRepositorySettingsHandlers_BadPayload(t *testing.T) {
	tempDir := t.TempDir()
	handler := NewRepositoryHandler(tempDir, nil, nil)
	repoName := "settings-repo"
	if _, err := createTestRepository(handler, repoName); err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		name   string
		method func(http.ResponseWriter, *http.Request)
		req    *http.Request
	}{
		{
			name:   "identity",
			method: handler.UpdateRepositorySettingsIdentity,
			req:    newRepoSettingsRequest(http.MethodPut, "/api/repos/"+repoName+"/settings/identity", repoName, []byte(`{"name":"","email":"invalid"}`)),
		},
		{
			name:   "sync",
			method: handler.UpdateRepositorySettingsSync,
			req:    newRepoSettingsRequest(http.MethodPut, "/api/repos/"+repoName+"/settings/sync", repoName, []byte(`{"autoFetch":true,"fetchIntervalMinutes":10,"pullStrategy":"merge"}`)),
		},
		{
			name:   "commit",
			method: handler.UpdateRepositorySettingsCommit,
			req:    newRepoSettingsRequest(http.MethodPut, "/api/repos/"+repoName+"/settings/commit", repoName, []byte(`{"defaultBranch":"","signingEnabled":false,"lineEndings":"tabs"}`)),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rec := httptest.NewRecorder()
			tt.method(rec, tt.req)
			if rec.Code != http.StatusBadRequest {
				t.Fatalf("expected 400, got %d: %s", rec.Code, rec.Body.String())
			}
		})
	}
}

func TestRepositorySettingsHandlers_RepoNotFound(t *testing.T) {
	tempDir := t.TempDir()
	handler := NewRepositoryHandler(tempDir, nil, nil)
	repoName := "missing-repo"

	tests := []struct {
		name   string
		method func(http.ResponseWriter, *http.Request)
		req    *http.Request
	}{
		{
			name:   "get",
			method: handler.GetRepositorySettings,
			req:    newRepoSettingsRequest(http.MethodGet, "/api/repos/"+repoName+"/settings", repoName, nil),
		},
		{
			name:   "identity",
			method: handler.UpdateRepositorySettingsIdentity,
			req:    newRepoSettingsRequest(http.MethodPut, "/api/repos/"+repoName+"/settings/identity", repoName, []byte(`{"name":"A","email":"a@example.com"}`)),
		},
		{
			name:   "sync",
			method: handler.UpdateRepositorySettingsSync,
			req:    newRepoSettingsRequest(http.MethodPut, "/api/repos/"+repoName+"/settings/sync", repoName, []byte(`{"autoFetch":true,"fetchIntervalMinutes":15,"pullStrategy":"merge"}`)),
		},
		{
			name:   "commit",
			method: handler.UpdateRepositorySettingsCommit,
			req:    newRepoSettingsRequest(http.MethodPut, "/api/repos/"+repoName+"/settings/commit", repoName, []byte(`{"defaultBranch":"main","signingEnabled":false,"lineEndings":"lf"}`)),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rec := httptest.NewRecorder()
			tt.method(rec, tt.req)
			if rec.Code != http.StatusNotFound {
				t.Fatalf("expected 404, got %d: %s", rec.Code, rec.Body.String())
			}
		})
	}
}
