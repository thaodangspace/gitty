package filesystem

import (
	"os"
	"path/filepath"
	"testing"
)

func TestNewService(t *testing.T) {
	service := NewService(true)
	if service == nil {
		t.Fatal("NewService() returned nil")
	}

	if !service.restrictToUserHome {
		t.Error("Service should restrict to user home when requested")
	}
}

func TestBrowseDirectory(t *testing.T) {
	service := NewService(false)
	
	tempDir, err := os.MkdirTemp("", "filesystem_test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tempDir)

	// Create some test files and directories
	testFiles := []string{
		"file1.txt",
		"file2.txt",
		"dir1",
		"dir2",
		".hidden_file",
		".hidden_dir",
	}

	for _, name := range testFiles {
		path := filepath.Join(tempDir, name)
		if name == "dir1" || name == "dir2" || name == ".hidden_dir" {
			err = os.Mkdir(path, 0755)
		} else {
			err = os.WriteFile(path, []byte("test content"), 0644)
		}
		if err != nil {
			t.Fatal(err)
		}
	}

	// Browse the directory
	listing, err := service.BrowseDirectory(tempDir)
	if err != nil {
		t.Fatal(err)
	}

	if listing == nil {
		t.Fatal("BrowseDirectory returned nil")
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

	// Check hidden files
	for _, entry := range listing.Entries {
		if entry.Name == ".hidden_file" || entry.Name == ".hidden_dir" {
			if !entry.IsHidden {
				t.Errorf("Expected %s to be hidden", entry.Name)
			}
		} else {
			if entry.IsHidden {
				t.Errorf("Expected %s to not be hidden", entry.Name)
			}
		}
	}
}

func TestBrowseDirectoryWithGitRepo(t *testing.T) {
	service := NewService(false)
	
	tempDir, err := os.MkdirTemp("", "filesystem_test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tempDir)

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

	// Browse the directory
	listing, err := service.BrowseDirectory(tempDir)
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

func TestBrowseDirectoryEmpty(t *testing.T) {
	service := NewService(false)
	
	tempDir, err := os.MkdirTemp("", "filesystem_test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tempDir)

	// Browse empty directory
	listing, err := service.BrowseDirectory(tempDir)
	if err != nil {
		t.Fatal(err)
	}

	if listing == nil {
		t.Fatal("BrowseDirectory returned nil")
	}

	if len(listing.Entries) != 0 {
		t.Errorf("Expected 0 entries, got %d", len(listing.Entries))
	}
}

func TestBrowseDirectoryNonExistent(t *testing.T) {
	service := NewService(false)
	
	nonExistentPath := "/path/that/does/not/exist"
	
	_, err := service.BrowseDirectory(nonExistentPath)
	if err == nil {
		t.Error("Expected error when browsing non-existent directory")
	}
}

func TestBrowseDirectoryNotDirectory(t *testing.T) {
	service := NewService(false)
	
	tempDir, err := os.MkdirTemp("", "filesystem_test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tempDir)

	// Create a file
	filePath := filepath.Join(tempDir, "test.txt")
	err = os.WriteFile(filePath, []byte("test content"), 0644)
	if err != nil {
		t.Fatal(err)
	}

	// Try to browse the file
	_, err = service.BrowseDirectory(filePath)
	if err == nil {
		t.Error("Expected error when browsing a file")
	}
}

func TestBrowseDirectoryWithParentPath(t *testing.T) {
	service := NewService(false)
	
	tempDir, err := os.MkdirTemp("", "filesystem_test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tempDir)

	// Create a subdirectory
	subDir := filepath.Join(tempDir, "subdir")
	err = os.Mkdir(subDir, 0755)
	if err != nil {
		t.Fatal(err)
	}

	// Browse the subdirectory
	listing, err := service.BrowseDirectory(subDir)
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

func TestIsPathAllowed(t *testing.T) {
	service := NewService(true)
	
	// Test with restricted service
	if !service.isPathAllowed(service.userHomeDir) {
		t.Error("User home directory should be allowed")
	}

	if service.isPathAllowed("/etc") {
		t.Error("System directory should not be allowed when restricted")
	}

	// Test with unrestricted service
	unrestrictedService := NewService(false)
	
	if !unrestrictedService.isPathAllowed("/etc") {
		t.Error("System directory should be allowed when not restricted")
	}
}

func TestCanGoUp(t *testing.T) {
	service := NewService(false)
	
	// Test normal case
	currentPath := "/home/user/project"
	parentPath := "/home/user"
	
	if !service.canGoUp(currentPath, parentPath) {
		t.Error("Should be able to go up normally")
	}

	// Test root case
	currentPath = "/"
	parentPath = "/"
	
	if service.canGoUp(currentPath, parentPath) {
		t.Error("Should not be able to go up from root")
	}

	// Test with restricted service
	restrictedService := NewService(true)
	
	if !restrictedService.canGoUp(restrictedService.userHomeDir+"/project", restrictedService.userHomeDir) {
		t.Error("Should be able to go up within user home")
	}

	if restrictedService.canGoUp("/etc", "/") {
		t.Error("Should not be able to go up outside user home when restricted")
	}
}

func TestIsHidden(t *testing.T) {
	service := NewService(false)
	
	testCases := []struct {
		name     string
		expected bool
	}{
		{".hidden", true},
		{"normal", false},
		{"file.txt", false},
		{".git", true},
		{".config", true},
	}

	for _, tc := range testCases {
		result := service.isHidden(tc.name)
		if result != tc.expected {
			t.Errorf("isHidden(%s) = %v, expected %v", tc.name, result, tc.expected)
		}
	}
}

func TestIsGitRepository(t *testing.T) {
	service := NewService(false)
	
	tempDir, err := os.MkdirTemp("", "filesystem_test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tempDir)

	// Test non-git directory
	if service.isGitRepository(tempDir) {
		t.Error("Empty directory should not be a git repository")
	}

	// Create .git directory
	gitDir := filepath.Join(tempDir, ".git")
	err = os.Mkdir(gitDir, 0755)
	if err != nil {
		t.Fatal(err)
	}

	// Test git directory
	if !service.isGitRepository(tempDir) {
		t.Error("Directory with .git should be a git repository")
	}
}

func TestGetVolumeRoots(t *testing.T) {
	service := NewService(false)
	
	roots, err := service.GetVolumeRoots()
	if err != nil {
		t.Fatal(err)
	}

	if len(roots) == 0 {
		t.Error("Should have at least one volume root")
	}

	// Check that all roots are directories
	for _, root := range roots {
		if !root.IsDirectory {
			t.Errorf("Volume root %s should be a directory", root.Name)
		}
	}
}

func TestBrowseDirectoryWithSpecialCharacters(t *testing.T) {
	service := NewService(false)
	
	tempDir, err := os.MkdirTemp("", "filesystem_test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tempDir)

	// Create files with special characters
	specialNames := []string{
		"file with spaces.txt",
		"file-with-dashes.txt",
		"file_with_underscores.txt",
		"file.with.dots.txt",
		"file@#$%^&*().txt",
	}

	for _, name := range specialNames {
		path := filepath.Join(tempDir, name)
		err = os.WriteFile(path, []byte("test content"), 0644)
		if err != nil {
			t.Fatal(err)
		}
	}

	// Browse the directory
	listing, err := service.BrowseDirectory(tempDir)
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

func TestBrowseDirectoryWithSymlinks(t *testing.T) {
	service := NewService(false)
	
	tempDir, err := os.MkdirTemp("", "filesystem_test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tempDir)

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

	// Browse the directory
	listing, err := service.BrowseDirectory(tempDir)
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

func TestBrowseDirectoryWithLargeFiles(t *testing.T) {
	service := NewService(false)
	
	tempDir, err := os.MkdirTemp("", "filesystem_test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tempDir)

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

	// Browse the directory
	listing, err := service.BrowseDirectory(tempDir)
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
	service := NewService(false)
	
	tempDir, err := os.MkdirTemp("", "filesystem_test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tempDir)

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

	// Browse the directory
	listing, err := service.BrowseDirectory(tempDir)
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