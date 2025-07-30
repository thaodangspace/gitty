package handlers

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"gitweb/server/internal/models"
	"github.com/go-chi/chi/v5"
	"context"
)

func TestNewFilesystemHandler(t *testing.T) {
	handler := NewFilesystemHandler(false)
	if handler == nil {
		t.Fatal("NewFilesystemHandler returned nil")
	}

	if handler.fsService == nil {
		t.Error("Filesystem service should be initialized")
	}
}

func TestBrowseDirectory(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "filesystem_handler_test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tempDir)

	handler := NewFilesystemHandler(false)

	// Create some test files and directories
	testFiles := []string{
		"file1.txt",
		"file2.txt",
		"dir1",
		"dir2",
	}

	for _, name := range testFiles {
		path := filepath.Join(tempDir, name)
		if name == "dir1" || name == "dir2" {
			err = os.Mkdir(path, 0755)
		} else {
			err = os.WriteFile(path, []byte("test content"), 0644)
		}
		if err != nil {
			t.Fatal(err)
		}
	}

	// Create HTTP request
	req := httptest.NewRequest("GET", "/filesystem/browse?path="+tempDir, nil)
	w := httptest.NewRecorder()

	// Call handler
	handler.BrowseDirectory(w, req)

	// Check response
	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	// Parse response
	var listing models.DirectoryListing
	err = json.Unmarshal(w.Body.Bytes(), &listing)
	if err != nil {
		t.Fatal(err)
	}

	if listing.CurrentPath != tempDir {
		t.Errorf("Expected current path %s, got %s", tempDir, listing.CurrentPath)
	}

	if len(listing.Entries) != len(testFiles) {
		t.Errorf("Expected %d entries, got %d", len(testFiles), len(listing.Entries))
	}

	// Verify entries
	foundFiles := make(map[string]bool)
	for _, entry := range listing.Entries {
		foundFiles[entry.Name] = true
	}

	for _, expectedFile := range testFiles {
		if !foundFiles[expectedFile] {
			t.Errorf("Expected file/directory %s not found", expectedFile)
		}
	}
}

func TestBrowseDirectoryWithPathParam(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "filesystem_handler_test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tempDir)

	handler := NewFilesystemHandler(false)

	// Create a test file
	testFile := filepath.Join(tempDir, "test.txt")
	err = os.WriteFile(testFile, []byte("test content"), 0644)
	if err != nil {
		t.Fatal(err)
	}

	// Create HTTP request with path parameter
	req := httptest.NewRequest("GET", "/filesystem/browse/"+tempDir, nil)
	w := httptest.NewRecorder()

	// Set up chi router context
	chiCtx := chi.NewRouteContext()
	chiCtx.URLParams.Add("*", tempDir)
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, chiCtx))

	// Call handler
	handler.BrowseDirectory(w, req)

	// Check response
	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	// Parse response
	var listing models.DirectoryListing
	err = json.Unmarshal(w.Body.Bytes(), &listing)
	if err != nil {
		t.Fatal(err)
	}

	if listing.CurrentPath != tempDir {
		t.Errorf("Expected current path %s, got %s", tempDir, listing.CurrentPath)
	}

	if len(listing.Entries) != 1 {
		t.Errorf("Expected 1 entry, got %d", len(listing.Entries))
	}

	if listing.Entries[0].Name != "test.txt" {
		t.Errorf("Expected file name 'test.txt', got '%s'", listing.Entries[0].Name)
	}
}

func TestBrowseDirectoryNonExistent(t *testing.T) {
	handler := NewFilesystemHandler(false)

	// Create HTTP request with non-existent path
	req := httptest.NewRequest("GET", "/filesystem/browse?path=/path/that/does/not/exist", nil)
	w := httptest.NewRecorder()

	// Call handler
	handler.BrowseDirectory(w, req)

	// Check response
	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400, got %d", w.Code)
	}
}

func TestBrowseDirectoryNotDirectory(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "filesystem_handler_test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tempDir)

	handler := NewFilesystemHandler(false)

	// Create a test file
	testFile := filepath.Join(tempDir, "test.txt")
	err = os.WriteFile(testFile, []byte("test content"), 0644)
	if err != nil {
		t.Fatal(err)
	}

	// Create HTTP request with file path
	req := httptest.NewRequest("GET", "/filesystem/browse?path="+testFile, nil)
	w := httptest.NewRecorder()

	// Call handler
	handler.BrowseDirectory(w, req)

	// Check response
	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400, got %d", w.Code)
	}
}

func TestBrowseDirectoryWithGitRepo(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "filesystem_handler_test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tempDir)

	handler := NewFilesystemHandler(false)

	// Create a .git directory to simulate a git repository
	gitDir := filepath.Join(tempDir, ".git")
	err = os.Mkdir(gitDir, 0755)
	if err != nil {
		t.Fatal(err)
	}

	// Create some files
	testFiles := []string{
		"file1.txt",
		"file2.txt",
	}

	for _, name := range testFiles {
		path := filepath.Join(tempDir, name)
		err = os.WriteFile(path, []byte("test content"), 0644)
		if err != nil {
			t.Fatal(err)
		}
	}

	// Create HTTP request
	req := httptest.NewRequest("GET", "/filesystem/browse?path="+tempDir, nil)
	w := httptest.NewRecorder()

	// Call handler
	handler.BrowseDirectory(w, req)

	// Check response
	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	// Parse response
	var listing models.DirectoryListing
	err = json.Unmarshal(w.Body.Bytes(), &listing)
	if err != nil {
		t.Fatal(err)
	}

	// Check that the directory is marked as a git repo
	for _, entry := range listing.Entries {
		if entry.Name == "." {
			if !entry.IsGitRepo {
				t.Error("Directory with .git should be marked as git repo")
			}
		}
	}
}

func TestBrowseDirectoryWithHiddenFiles(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "filesystem_handler_test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tempDir)

	handler := NewFilesystemHandler(false)

	// Create hidden files
	hiddenFiles := []string{
		".hidden_file",
		".hidden_dir",
	}

	for _, name := range hiddenFiles {
		path := filepath.Join(tempDir, name)
		if name == ".hidden_dir" {
			err = os.Mkdir(path, 0755)
		} else {
			err = os.WriteFile(path, []byte("test content"), 0644)
		}
		if err != nil {
			t.Fatal(err)
		}
	}

	// Create HTTP request
	req := httptest.NewRequest("GET", "/filesystem/browse?path="+tempDir, nil)
	w := httptest.NewRecorder()

	// Call handler
	handler.BrowseDirectory(w, req)

	// Check response
	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	// Parse response
	var listing models.DirectoryListing
	err = json.Unmarshal(w.Body.Bytes(), &listing)
	if err != nil {
		t.Fatal(err)
	}

	// Check that hidden files are marked correctly
	for _, entry := range listing.Entries {
		if entry.Name == ".hidden_file" || entry.Name == ".hidden_dir" {
			if !entry.IsHidden {
				t.Errorf("Expected %s to be hidden", entry.Name)
			}
		}
	}
}

func TestBrowseDirectoryWithSpecialCharacters(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "filesystem_handler_test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tempDir)

	handler := NewFilesystemHandler(false)

	// Create files with special characters
	specialNames := []string{
		"file with spaces.txt",
		"file-with-dashes.txt",
		"file_with_underscores.txt",
		"file.with.dots.txt",
	}

	for _, name := range specialNames {
		path := filepath.Join(tempDir, name)
		err = os.WriteFile(path, []byte("test content"), 0644)
		if err != nil {
			t.Fatal(err)
		}
	}

	// Create HTTP request
	req := httptest.NewRequest("GET", "/filesystem/browse?path="+tempDir, nil)
	w := httptest.NewRecorder()

	// Call handler
	handler.BrowseDirectory(w, req)

	// Check response
	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	// Parse response
	var listing models.DirectoryListing
	err = json.Unmarshal(w.Body.Bytes(), &listing)
	if err != nil {
		t.Fatal(err)
	}

	if len(listing.Entries) != len(specialNames) {
		t.Errorf("Expected %d entries, got %d", len(specialNames), len(listing.Entries))
	}

	// Verify all special files are present
	foundFiles := make(map[string]bool)
	for _, entry := range listing.Entries {
		foundFiles[entry.Name] = true
	}

	for _, expectedFile := range specialNames {
		if !foundFiles[expectedFile] {
			t.Errorf("Expected file %s not found", expectedFile)
		}
	}
}

func TestBrowseDirectoryWithParentPath(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "filesystem_handler_test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tempDir)

	handler := NewFilesystemHandler(false)

	// Create a subdirectory
	subDir := filepath.Join(tempDir, "subdir")
	err = os.Mkdir(subDir, 0755)
	if err != nil {
		t.Fatal(err)
	}

	// Create HTTP request
	req := httptest.NewRequest("GET", "/filesystem/browse?path="+subDir, nil)
	w := httptest.NewRecorder()

	// Call handler
	handler.BrowseDirectory(w, req)

	// Check response
	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	// Parse response
	var listing models.DirectoryListing
	err = json.Unmarshal(w.Body.Bytes(), &listing)
	if err != nil {
		t.Fatal(err)
	}

	if listing.ParentPath != tempDir {
		t.Errorf("Expected parent path %s, got %s", tempDir, listing.ParentPath)
	}

	if !listing.CanGoUp {
		t.Error("Should be able to go up from subdirectory")
	}
}

func TestBrowseDirectoryEmpty(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "filesystem_handler_test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tempDir)

	handler := NewFilesystemHandler(false)

	// Create HTTP request
	req := httptest.NewRequest("GET", "/filesystem/browse?path="+tempDir, nil)
	w := httptest.NewRecorder()

	// Call handler
	handler.BrowseDirectory(w, req)

	// Check response
	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	// Parse response
	var listing models.DirectoryListing
	err = json.Unmarshal(w.Body.Bytes(), &listing)
	if err != nil {
		t.Fatal(err)
	}

	if len(listing.Entries) != 0 {
		t.Errorf("Expected 0 entries, got %d", len(listing.Entries))
	}
}

func TestBrowseDirectoryWithLargeFiles(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "filesystem_handler_test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tempDir)

	handler := NewFilesystemHandler(false)

	// Create a large file (1MB)
	largeFile := filepath.Join(tempDir, "large.txt")
	largeContent := make([]byte, 1024*1024) // 1MB
	for i := range largeContent {
		largeContent[i] = byte(i % 256)
	}
	
	err = os.WriteFile(largeFile, largeContent, 0644)
	if err != nil {
		t.Fatal(err)
	}

	// Create HTTP request
	req := httptest.NewRequest("GET", "/filesystem/browse?path="+tempDir, nil)
	w := httptest.NewRecorder()

	// Call handler
	handler.BrowseDirectory(w, req)

	// Check response
	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	// Parse response
	var listing models.DirectoryListing
	err = json.Unmarshal(w.Body.Bytes(), &listing)
	if err != nil {
		t.Fatal(err)
	}

	if len(listing.Entries) != 1 {
		t.Errorf("Expected 1 entry, got %d", len(listing.Entries))
	}

	entry := listing.Entries[0]
	if entry.Name != "large.txt" {
		t.Errorf("Expected file name 'large.txt', got '%s'", entry.Name)
	}

	if entry.Size != int64(len(largeContent)) {
		t.Errorf("Expected file size %d, got %d", len(largeContent), entry.Size)
	}
}

func TestBrowseDirectoryWithPermissions(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "filesystem_handler_test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tempDir)

	handler := NewFilesystemHandler(false)

	// Create files with different permissions
	testFiles := []struct {
		name string
		mode os.FileMode
	}{
		{"readonly.txt", 0444},
		{"executable.txt", 0755},
		{"normal.txt", 0644},
	}

	for _, tc := range testFiles {
		path := filepath.Join(tempDir, tc.name)
		err = os.WriteFile(path, []byte("test content"), tc.mode)
		if err != nil {
			t.Fatal(err)
		}
	}

	// Create HTTP request
	req := httptest.NewRequest("GET", "/filesystem/browse?path="+tempDir, nil)
	w := httptest.NewRecorder()

	// Call handler
	handler.BrowseDirectory(w, req)

	// Check response
	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	// Parse response
	var listing models.DirectoryListing
	err = json.Unmarshal(w.Body.Bytes(), &listing)
	if err != nil {
		t.Fatal(err)
	}

	if len(listing.Entries) != len(testFiles) {
		t.Errorf("Expected %d entries, got %d", len(testFiles), len(listing.Entries))
	}

	// Verify permissions are set correctly
	for _, entry := range listing.Entries {
		if entry.Permissions == "" {
			t.Errorf("Entry %s should have permissions set", entry.Name)
		}
	}
}

func TestBrowseDirectoryWithSymlinks(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "filesystem_handler_test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tempDir)

	handler := NewFilesystemHandler(false)

	// Create a target file
	targetFile := filepath.Join(tempDir, "target.txt")
	err = os.WriteFile(targetFile, []byte("target content"), 0644)
	if err != nil {
		t.Fatal(err)
	}

	// Create a symlink
	symlinkFile := filepath.Join(tempDir, "symlink.txt")
	err = os.Symlink(targetFile, symlinkFile)
	if err != nil {
		// Skip test if symlinks are not supported
		t.Skip("Symlinks not supported on this system")
	}

	// Create HTTP request
	req := httptest.NewRequest("GET", "/filesystem/browse?path="+tempDir, nil)
	w := httptest.NewRecorder()

	// Call handler
	handler.BrowseDirectory(w, req)

	// Check response
	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	// Parse response
	var listing models.DirectoryListing
	err = json.Unmarshal(w.Body.Bytes(), &listing)
	if err != nil {
		t.Fatal(err)
	}

	// Should find both the target and symlink
	if len(listing.Entries) != 2 {
		t.Errorf("Expected 2 entries, got %d", len(listing.Entries))
	}

	foundTarget := false
	foundSymlink := false
	for _, entry := range listing.Entries {
		if entry.Name == "target.txt" {
			foundTarget = true
		}
		if entry.Name == "symlink.txt" {
			foundSymlink = true
		}
	}

	if !foundTarget {
		t.Error("Target file not found")
	}
	if !foundSymlink {
		t.Error("Symlink not found")
	}
}

func TestBrowseDirectoryRestricted(t *testing.T) {
	handler := NewFilesystemHandler(true)

	// Try to browse a system directory
	req := httptest.NewRequest("GET", "/filesystem/browse?path=/etc", nil)
	w := httptest.NewRecorder()

	// Call handler
	handler.BrowseDirectory(w, req)

	// Check response - should be denied
	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400, got %d", w.Code)
	}
}

func TestBrowseDirectoryUnrestricted(t *testing.T) {
	handler := NewFilesystemHandler(false)

	// Try to browse a system directory
	req := httptest.NewRequest("GET", "/filesystem/browse?path=/etc", nil)
	w := httptest.NewRecorder()

	// Call handler
	handler.BrowseDirectory(w, req)

	// Check response - should be allowed
	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}
}