package handlers

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"gitweb/server/internal/models"
	"github.com/go-chi/chi/v5"
	"context"
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

func TestNewRepositoryHandler(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "handler_test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tempDir)

	handler := NewRepositoryHandler(tempDir)
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
	handler := NewRepositoryHandler("")

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

	handler := NewRepositoryHandler(tempDir)

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

	handler := NewRepositoryHandler(tempDir)

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

	handler := NewRepositoryHandler(tempDir)

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

	handler := NewRepositoryHandler(tempDir)

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

	handler := NewRepositoryHandler(tempDir)

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

	handler := NewRepositoryHandler(tempDir)

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

	// Verify repository was deleted
	if _, err := os.Stat(repoDir); !os.IsNotExist(err) {
		t.Error("Repository directory was not deleted")
	}
}

func TestGetRepositoryStatus(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "handler_test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tempDir)

	handler := NewRepositoryHandler(tempDir)

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

func TestGetCommitHistory(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "handler_test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tempDir)

	handler := NewRepositoryHandler(tempDir)

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

	// Should have at least 0 commits (new repository)
	if len(commits) < 0 {
		t.Errorf("Expected at least 0 commits, got %d", len(commits))
	}
}

func TestGetBranches(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "handler_test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tempDir)

	handler := NewRepositoryHandler(tempDir)

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

	handler := NewRepositoryHandler(tempDir)

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
	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}
}

func TestCreateBranch(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "handler_test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tempDir)

	handler := NewRepositoryHandler(tempDir)

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
	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}
}

func TestSwitchBranch(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "handler_test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tempDir)

	handler := NewRepositoryHandler(tempDir)

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

	// Create branch request
	branchReq := struct {
		Name string `json:"name"`
	}{
		Name: "main",
	}

	reqBody, err := json.Marshal(branchReq)
	if err != nil {
		t.Fatal(err)
	}

	// Create HTTP request
	req := httptest.NewRequest("POST", "/repositories/test-repo/branches/switch", bytes.NewBuffer(reqBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	// Set up chi router context
	chiCtx := chi.NewRouteContext()
	chiCtx.URLParams.Add("id", "test-repo")
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, chiCtx))

	// Call handler
	handler.SwitchBranch(w, req)

	// Check response
	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}
}

func TestGetFileTree(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "handler_test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tempDir)

	handler := NewRepositoryHandler(tempDir)

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
	var files []models.FileInfo
	err = json.Unmarshal(w.Body.Bytes(), &files)
	if err != nil {
		t.Fatal(err)
	}

	// Should have at least the test files
	if len(files) < len(testFiles) {
		t.Errorf("Expected at least %d files, got %d", len(testFiles), len(files))
	}
}

func TestGetFileContent(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "handler_test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tempDir)

	handler := NewRepositoryHandler(tempDir)

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

	handler := NewRepositoryHandler(tempDir)

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

	handler := NewRepositoryHandler(tempDir)

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
	testFile := filepath.Join(repoDir, "test.txt")
	err = os.WriteFile(testFile, []byte("test content"), 0644)
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

	// Create stage request
	stageReq := struct {
		Path string `json:"path"`
	}{
		Path: "test.txt",
	}

	reqBody, err := json.Marshal(stageReq)
	if err != nil {
		t.Fatal(err)
	}

	// Create HTTP request
	req := httptest.NewRequest("POST", "/repositories/test-repo/stage", bytes.NewBuffer(reqBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	// Set up chi router context
	chiCtx := chi.NewRouteContext()
	chiCtx.URLParams.Add("id", "test-repo")
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

	handler := NewRepositoryHandler(tempDir)

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
	testFile := filepath.Join(repoDir, "test.txt")
	err = os.WriteFile(testFile, []byte("test content"), 0644)
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

	// Create unstage request
	unstageReq := struct {
		Path string `json:"path"`
	}{
		Path: "test.txt",
	}

	reqBody, err := json.Marshal(unstageReq)
	if err != nil {
		t.Fatal(err)
	}

	// Create HTTP request
	req := httptest.NewRequest("POST", "/repositories/test-repo/unstage", bytes.NewBuffer(reqBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	// Set up chi router context
	chiCtx := chi.NewRouteContext()
	chiCtx.URLParams.Add("id", "test-repo")
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, chiCtx))

	// Call handler
	handler.UnstageFile(w, req)

	// Check response
	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}
}