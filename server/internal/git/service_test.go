package git

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"gitweb/server/internal/models"
)

func TestNewService(t *testing.T) {
	service := NewService()
	if service == nil {
		t.Fatal("NewService() returned nil")
	}
}

func TestOpenRepository(t *testing.T) {
	service := NewService()
	
	// Create a temporary directory for testing
	tempDir, err := os.MkdirTemp("", "git_test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tempDir)

	// Test opening a non-existent repository
	_, err = service.OpenRepository(tempDir)
	if err == nil {
		t.Error("Expected error when opening non-existent repository")
	}

	// Initialize a git repository
	repo, err := service.InitRepository(tempDir)
	if err != nil {
		t.Fatal(err)
	}

	// Test opening the initialized repository
	openedRepo, err := service.OpenRepository(tempDir)
	if err != nil {
		t.Fatal(err)
	}

	if openedRepo == nil {
		t.Fatal("OpenRepository returned nil")
	}

	// Verify it's the same repository
	head1, _ := repo.Head()
	head2, _ := openedRepo.Head()
	if head1 != nil && head2 != nil && head1.Hash() != head2.Hash() {
		t.Error("Repository heads don't match")
	}
}

func TestInitRepository(t *testing.T) {
	service := NewService()
	
	tempDir, err := os.MkdirTemp("", "git_test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tempDir)

	repo, err := service.InitRepository(tempDir)
	if err != nil {
		t.Fatal(err)
	}

	if repo == nil {
		t.Fatal("InitRepository returned nil")
	}

	// Verify .git directory exists
	gitPath := filepath.Join(tempDir, ".git")
	if _, err := os.Stat(gitPath); os.IsNotExist(err) {
		t.Error(".git directory was not created")
	}
}

func TestGetRepositoryStatus(t *testing.T) {
	service := NewService()
	
	tempDir, err := os.MkdirTemp("", "git_test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tempDir)

	// Initialize repository
	_, err = service.InitRepository(tempDir)
	if err != nil {
		t.Fatal(err)
	}

	// Create initial commit to establish HEAD
	testFile := filepath.Join(tempDir, "README.md")
	err = os.WriteFile(testFile, []byte("# Test Repository"), 0644)
	if err != nil {
		t.Fatal(err)
	}

	// Stage and commit the file
	err = service.StageFile(tempDir, "README.md")
	if err != nil {
		t.Fatal(err)
	}

	commitReq := models.CommitRequest{
		Message: "Initial commit",
		Files:   []string{"README.md"},
		Author: models.Author{
			Name:  "Test User",
			Email: "test@example.com",
		},
	}

	err = service.CreateCommit(tempDir, commitReq)
	if err != nil {
		t.Fatal(err)
	}

	// Get status
	status, err := service.GetRepositoryStatus(tempDir)
	if err != nil {
		t.Fatal(err)
	}

	if status == nil {
		t.Fatal("GetRepositoryStatus returned nil")
	}

	// Verify initial state
	if !status.IsClean {
		t.Error("Repository with no changes should be clean")
	}

	if len(status.Staged) != 0 {
		t.Error("Repository should have no staged files")
	}

	if len(status.Modified) != 0 {
		t.Error("Repository should have no modified files")
	}

	if len(status.Untracked) != 0 {
		t.Error("Repository should have no untracked files")
	}
}

func TestGetBranches(t *testing.T) {
	service := NewService()
	
	tempDir, err := os.MkdirTemp("", "git_test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tempDir)

	// Initialize repository
	_, err = service.InitRepository(tempDir)
	if err != nil {
		t.Fatal(err)
	}

	// Create initial commit to establish HEAD
	testFile := filepath.Join(tempDir, "README.md")
	err = os.WriteFile(testFile, []byte("# Test Repository"), 0644)
	if err != nil {
		t.Fatal(err)
	}

	// Stage and commit the file
	err = service.StageFile(tempDir, "README.md")
	if err != nil {
		t.Fatal(err)
	}

	commitReq := models.CommitRequest{
		Message: "Initial commit",
		Files:   []string{"README.md"},
		Author: models.Author{
			Name:  "Test User",
			Email: "test@example.com",
		},
	}

	err = service.CreateCommit(tempDir, commitReq)
	if err != nil {
		t.Fatal(err)
	}

	// Get branches
	branches, err := service.GetBranches(tempDir)
	if err != nil {
		t.Fatal(err)
	}

	if len(branches) == 0 {
		t.Error("Should have at least one branch (main/master)")
	}

	// Check that we have a main or master branch
	foundMainBranch := false
	for _, branch := range branches {
		if branch.Name == "main" || branch.Name == "master" {
			foundMainBranch = true
			if !branch.IsCurrent {
				t.Error("Main branch should be current")
			}
			break
		}
	}

	if !foundMainBranch {
		t.Error("Should have main or master branch")
	}
}

func TestCreateBranch(t *testing.T) {
	service := NewService()
	
	tempDir, err := os.MkdirTemp("", "git_test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tempDir)

	// Initialize repository
	_, err = service.InitRepository(tempDir)
	if err != nil {
		t.Fatal(err)
	}

	// Create initial commit to establish HEAD
	testFile := filepath.Join(tempDir, "README.md")
	err = os.WriteFile(testFile, []byte("# Test Repository"), 0644)
	if err != nil {
		t.Fatal(err)
	}

	// Stage and commit the file
	err = service.StageFile(tempDir, "README.md")
	if err != nil {
		t.Fatal(err)
	}

	commitReq := models.CommitRequest{
		Message: "Initial commit",
		Files:   []string{"README.md"},
		Author: models.Author{
			Name:  "Test User",
			Email: "test@example.com",
		},
	}

	err = service.CreateCommit(tempDir, commitReq)
	if err != nil {
		t.Fatal(err)
	}

	// Create a new branch
	branchName := "test-branch"
	err = service.CreateBranch(tempDir, branchName)
	if err != nil {
		t.Fatal(err)
	}

	// Verify branch was created
	branches, err := service.GetBranches(tempDir)
	if err != nil {
		t.Fatal(err)
	}

	foundBranch := false
	for _, branch := range branches {
		if branch.Name == branchName {
			foundBranch = true
			break
		}
	}

	if !foundBranch {
		t.Error("Created branch not found in branch list")
	}
}

func TestSwitchBranch(t *testing.T) {
	service := NewService()
	
	tempDir, err := os.MkdirTemp("", "git_test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tempDir)

	// Initialize repository
	_, err = service.InitRepository(tempDir)
	if err != nil {
		t.Fatal(err)
	}

	// Create initial commit to establish HEAD
	testFile := filepath.Join(tempDir, "README.md")
	err = os.WriteFile(testFile, []byte("# Test Repository"), 0644)
	if err != nil {
		t.Fatal(err)
	}

	// Stage and commit the file
	err = service.StageFile(tempDir, "README.md")
	if err != nil {
		t.Fatal(err)
	}

	commitReq := models.CommitRequest{
		Message: "Initial commit",
		Files:   []string{"README.md"},
		Author: models.Author{
			Name:  "Test User",
			Email: "test@example.com",
		},
	}

	err = service.CreateCommit(tempDir, commitReq)
	if err != nil {
		t.Fatal(err)
	}

	// Create a new branch
	branchName := "test-branch"
	err = service.CreateBranch(tempDir, branchName)
	if err != nil {
		t.Fatal(err)
	}

	// Switch to the new branch
	err = service.SwitchBranch(tempDir, branchName)
	if err != nil {
		t.Fatal(err)
	}

	// Verify we're on the new branch
	status, err := service.GetRepositoryStatus(tempDir)
	if err != nil {
		t.Fatal(err)
	}

	if status.Branch != branchName {
		t.Errorf("Expected to be on branch %s, but on %s", branchName, status.Branch)
	}
}

func TestGetFileContentAndSaveFileContent(t *testing.T) {
	service := NewService()
	
	tempDir, err := os.MkdirTemp("", "git_test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tempDir)

	// Initialize repository
	_, err = service.InitRepository(tempDir)
	if err != nil {
		t.Fatal(err)
	}

	// Create a test file
	testContent := []byte("Hello, World!")
	filePath := "test.txt"
	
	err = service.SaveFileContent(tempDir, filePath, testContent)
	if err != nil {
		t.Fatal(err)
	}

	// Read the file content
	content, err := service.GetFileContent(tempDir, filePath)
	if err != nil {
		t.Fatal(err)
	}

	if string(content) != string(testContent) {
		t.Errorf("Expected content %s, got %s", string(testContent), string(content))
	}
}

func TestStageAndUnstageFile(t *testing.T) {
	service := NewService()
	
	tempDir, err := os.MkdirTemp("", "git_test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tempDir)

	// Initialize repository
	_, err = service.InitRepository(tempDir)
	if err != nil {
		t.Fatal(err)
	}

	// Create initial commit to establish HEAD
	testFile := filepath.Join(tempDir, "README.md")
	err = os.WriteFile(testFile, []byte("# Test Repository"), 0644)
	if err != nil {
		t.Fatal(err)
	}

	// Stage and commit the file
	err = service.StageFile(tempDir, "README.md")
	if err != nil {
		t.Fatal(err)
	}

	commitReq := models.CommitRequest{
		Message: "Initial commit",
		Files:   []string{"README.md"},
		Author: models.Author{
			Name:  "Test User",
			Email: "test@example.com",
		},
	}

	err = service.CreateCommit(tempDir, commitReq)
	if err != nil {
		t.Fatal(err)
	}

	// Create a test file
	testContent := []byte("Hello, World!")
	filePath := "test.txt"
	
	err = service.SaveFileContent(tempDir, filePath, testContent)
	if err != nil {
		t.Fatal(err)
	}

	// Check initial status (should be untracked)
	status, err := service.GetRepositoryStatus(tempDir)
	if err != nil {
		t.Fatal(err)
	}

	if len(status.Untracked) == 0 {
		t.Error("File should be untracked initially")
	}

	// Stage the file
	err = service.StageFile(tempDir, filePath)
	if err != nil {
		t.Fatal(err)
	}

	// Check status after staging
	status, err = service.GetRepositoryStatus(tempDir)
	if err != nil {
		t.Fatal(err)
	}

	if len(status.Staged) == 0 {
		t.Error("File should be staged")
	}

	// Unstage the file
	err = service.UnstageFile(tempDir, filePath)
	if err != nil {
		t.Fatal(err)
	}

	// Check status after unstaging
	status, err = service.GetRepositoryStatus(tempDir)
	if err != nil {
		t.Fatal(err)
	}

	// After unstaging, the file should be back in untracked or modified
	// The exact behavior depends on the git implementation
	// For now, just check that it's not in staged
	if len(status.Staged) != 0 {
		t.Error("File should not be staged after unstaging")
	}
}

func TestCreateCommit(t *testing.T) {
	service := NewService()
	
	tempDir, err := os.MkdirTemp("", "git_test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tempDir)

	// Initialize repository
	_, err = service.InitRepository(tempDir)
	if err != nil {
		t.Fatal(err)
	}

	// Create a test file
	testContent := []byte("Hello, World!")
	filePath := "test.txt"
	
	err = service.SaveFileContent(tempDir, filePath, testContent)
	if err != nil {
		t.Fatal(err)
	}

	// Stage the file
	err = service.StageFile(tempDir, filePath)
	if err != nil {
		t.Fatal(err)
	}

	// Create a commit
	commitReq := models.CommitRequest{
		Message: "Test commit",
		Files:   []string{filePath},
		Author: models.Author{
			Name:  "Test User",
			Email: "test@example.com",
		},
	}

	err = service.CreateCommit(tempDir, commitReq)
	if err != nil {
		t.Fatal(err)
	}

	// Verify commit was created
	commits, err := service.GetCommitHistory(tempDir, 10)
	if err != nil {
		t.Fatal(err)
	}

	if len(commits) == 0 {
		t.Error("Should have at least one commit")
	}

	latestCommit := commits[0]
	if latestCommit.Message != "Test commit" {
		t.Errorf("Expected commit message 'Test commit', got '%s'", latestCommit.Message)
	}

	if latestCommit.Author.Name != "Test User" {
		t.Errorf("Expected author name 'Test User', got '%s'", latestCommit.Author.Name)
	}
}

func TestGetCommitHistory(t *testing.T) {
	service := NewService()
	
	tempDir, err := os.MkdirTemp("", "git_test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tempDir)

	// Initialize repository
	_, err = service.InitRepository(tempDir)
	if err != nil {
		t.Fatal(err)
	}

	// Create multiple commits
	for i := 0; i < 3; i++ {
		// Create a test file
		testContent := []byte(fmt.Sprintf("Content %d", i))
		filePath := fmt.Sprintf("test%d.txt", i)
		
		err = service.SaveFileContent(tempDir, filePath, testContent)
		if err != nil {
			t.Fatal(err)
		}

		// Stage the file
		err = service.StageFile(tempDir, filePath)
		if err != nil {
			t.Fatal(err)
		}

		// Create a commit
		commitReq := models.CommitRequest{
			Message: fmt.Sprintf("Commit %d", i),
			Files:   []string{filePath},
			Author: models.Author{
				Name:  "Test User",
				Email: "test@example.com",
			},
		}

		err = service.CreateCommit(tempDir, commitReq)
		if err != nil {
			t.Fatal(err)
		}
	}

	// Get commit history
	commits, err := service.GetCommitHistory(tempDir, 10)
	if err != nil {
		t.Fatal(err)
	}

	if len(commits) != 3 {
		t.Errorf("Expected 3 commits, got %d", len(commits))
	}

	// Verify commit messages
	for i, commit := range commits {
		expectedMessage := fmt.Sprintf("Commit %d", 2-i) // Commits are in reverse order
		if commit.Message != expectedMessage {
			t.Errorf("Expected commit message '%s', got '%s'", expectedMessage, commit.Message)
		}
	}
}

func TestGetFileTree(t *testing.T) {
	service := NewService()
	
	tempDir, err := os.MkdirTemp("", "git_test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tempDir)

	// Initialize repository
	_, err = service.InitRepository(tempDir)
	if err != nil {
		t.Fatal(err)
	}

	// Create some test files and directories
	testFiles := []string{
		"file1.txt",
		"dir1/file2.txt",
		"dir1/dir2/file3.txt",
	}

	for _, filePath := range testFiles {
		// Create directory if needed
		dir := filepath.Dir(filePath)
		if dir != "." {
			err = os.MkdirAll(filepath.Join(tempDir, dir), 0755)
			if err != nil {
				t.Fatal(err)
			}
		}

		// Create file
		testContent := []byte("Test content")
		err = service.SaveFileContent(tempDir, filePath, testContent)
		if err != nil {
			t.Fatal(err)
		}
	}

	// Get file tree
	fileTree, err := service.GetFileTree(tempDir)
	if err != nil {
		t.Fatal(err)
	}

	if len(fileTree) == 0 {
		t.Error("File tree should not be empty")
	}

	// Verify we have the expected files
	foundFiles := make(map[string]bool)
	for _, file := range fileTree {
		foundFiles[file.Path] = true
	}

	for _, expectedFile := range testFiles {
		if !foundFiles[expectedFile] {
			t.Errorf("Expected file %s not found in file tree", expectedFile)
		}
	}
}

func TestDeleteBranch(t *testing.T) {
	service := NewService()
	
	tempDir, err := os.MkdirTemp("", "git_test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tempDir)

	// Initialize repository
	_, err = service.InitRepository(tempDir)
	if err != nil {
		t.Fatal(err)
	}

	// Create initial commit to establish HEAD
	testFile := filepath.Join(tempDir, "README.md")
	err = os.WriteFile(testFile, []byte("# Test Repository"), 0644)
	if err != nil {
		t.Fatal(err)
	}

	// Stage and commit the file
	err = service.StageFile(tempDir, "README.md")
	if err != nil {
		t.Fatal(err)
	}

	commitReq := models.CommitRequest{
		Message: "Initial commit",
		Files:   []string{"README.md"},
		Author: models.Author{
			Name:  "Test User",
			Email: "test@example.com",
		},
	}

	err = service.CreateCommit(tempDir, commitReq)
	if err != nil {
		t.Fatal(err)
	}

	// Create a new branch
	branchName := "test-branch"
	err = service.CreateBranch(tempDir, branchName)
	if err != nil {
		t.Fatal(err)
	}

	// Verify branch exists
	branches, err := service.GetBranches(tempDir)
	if err != nil {
		t.Fatal(err)
	}

	foundBranch := false
	for _, branch := range branches {
		if branch.Name == branchName {
			foundBranch = true
			break
		}
	}

	if !foundBranch {
		t.Error("Created branch not found")
	}

	// Delete the branch
	err = service.DeleteBranch(tempDir, branchName)
	if err != nil {
		t.Fatal(err)
	}

	// Verify branch was deleted
	branches, err = service.GetBranches(tempDir)
	if err != nil {
		t.Fatal(err)
	}

	foundBranch = false
	for _, branch := range branches {
		if branch.Name == branchName {
			foundBranch = true
			break
		}
	}

	if foundBranch {
		t.Error("Branch should have been deleted")
	}
}

func TestGetFileDiff(t *testing.T) {
	service := NewService()
	
	tempDir, err := os.MkdirTemp("", "git_test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tempDir)

	// Initialize repository
	_, err = service.InitRepository(tempDir)
	if err != nil {
		t.Fatal(err)
	}

	// Create initial file
	filePath := "test.txt"
	initialContent := []byte("Hello, World!")
	
	err = service.SaveFileContent(tempDir, filePath, initialContent)
	if err != nil {
		t.Fatal(err)
	}

	// Stage and commit initial file
	err = service.StageFile(tempDir, filePath)
	if err != nil {
		t.Fatal(err)
	}

	commitReq := models.CommitRequest{
		Message: "Initial commit",
		Files:   []string{filePath},
		Author: models.Author{
			Name:  "Test User",
			Email: "test@example.com",
		},
	}

	err = service.CreateCommit(tempDir, commitReq)
	if err != nil {
		t.Fatal(err)
	}

	// Modify the file
	modifiedContent := []byte("Hello, Modified World!")
	err = service.SaveFileContent(tempDir, filePath, modifiedContent)
	if err != nil {
		t.Fatal(err)
	}

	// Get file diff
	diff, err := service.GetFileDiff(tempDir, filePath)
	if err != nil {
		t.Fatal(err)
	}

	if diff == "" {
		t.Error("Expected non-empty diff")
	}

	// Verify diff contains expected content
	if !strings.Contains(diff, "Hello, World!") {
		t.Error("Diff should contain original content")
	}

	if !strings.Contains(diff, "Hello, Modified World!") {
		t.Error("Diff should contain modified content")
	}
}