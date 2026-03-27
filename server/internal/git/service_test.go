package git

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	gogitconfig "github.com/go-git/go-git/v5/config"
	"gitweb/server/internal/models"
)

func containsFileChangePath(changes []models.FileChange, path string) bool {
	for _, change := range changes {
		if change.Path == path {
			return true
		}
	}

	return false
}

func containsFileChangeStatus(changes []models.FileChange, status string) bool {
	for _, change := range changes {
		if change.Status == status {
			return true
		}
	}

	return false
}

func containsString(values []string, value string) bool {
	for _, current := range values {
		if current == value {
			return true
		}
	}

	return false
}

func stagedStatusesOnlyAllowlisted(changes []models.FileChange) bool {
	allowlist := map[string]struct{}{
		"A": {},
		"M": {},
		"D": {},
		"R": {},
		"C": {},
	}

	for _, change := range changes {
		if _, ok := allowlist[change.Status]; !ok {
			return false
		}
	}

	return true
}

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

func TestSetGitConfigIdentity(t *testing.T) {
	service := NewService()

	tempDir, err := os.MkdirTemp("", "git_test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tempDir)

	_, err = service.InitRepository(tempDir)
	if err != nil {
		t.Fatal(err)
	}

	if err := service.SetGitConfigIdentity(tempDir, "Alice Example", "alice@example.com"); err != nil {
		t.Fatal(err)
	}

	gitConfig, err := service.GetGitConfig(tempDir)
	if err != nil {
		t.Fatal(err)
	}

	if gitConfig.Name != "Alice Example" || gitConfig.Email != "alice@example.com" {
		t.Fatalf("unexpected git config: %+v", gitConfig)
	}
}

func TestGetRemotes(t *testing.T) {
	service := NewService()

	tempDir, err := os.MkdirTemp("", "git_test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tempDir)

	_, err = service.InitRepository(tempDir)
	if err != nil {
		t.Fatal(err)
	}

	repo, err := service.OpenRepository(tempDir)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := repo.CreateRemote(&gogitconfig.RemoteConfig{
		Name: "origin",
		URLs: []string{"https://example.com/repo.git"},
	}); err != nil {
		t.Fatal(err)
	}

	remotes, err := service.GetRemotes(tempDir)
	if err != nil {
		t.Fatal(err)
	}

	if len(remotes) != 1 {
		t.Fatalf("expected 1 remote, got %d", len(remotes))
	}
	if remotes[0].Name != "origin" || remotes[0].URL != "https://example.com/repo.git" {
		t.Fatalf("unexpected remote: %+v", remotes[0])
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

	if containsFileChangeStatus(status.Staged, "?") {
		t.Error("Untracked file should not appear in staged entries with status '?'")
	}

	if containsFileChangePath(status.Staged, filePath) {
		t.Error("Untracked file should not appear in staged entries")
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

func TestGetRepositoryStatus_StrictNormalization(t *testing.T) {
	service := NewService()

	tempDir, err := os.MkdirTemp("", "git_test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tempDir)

	_, err = service.InitRepository(tempDir)
	if err != nil {
		t.Fatal(err)
	}

	trackedPath := "README.md"
	if err := service.SaveFileContent(tempDir, trackedPath, []byte("# Test Repository")); err != nil {
		t.Fatal(err)
	}
	if err := service.StageFile(tempDir, trackedPath); err != nil {
		t.Fatal(err)
	}
	if err := service.CreateCommit(tempDir, models.CommitRequest{
		Message: "Initial commit",
		Files:   []string{trackedPath},
		Author: models.Author{
			Name:  "Test User",
			Email: "test@example.com",
		},
	}); err != nil {
		t.Fatal(err)
	}

	if err := service.SaveFileContent(tempDir, trackedPath, []byte("# Test Repository\nstaged")); err != nil {
		t.Fatal(err)
	}
	if err := service.StageFile(tempDir, trackedPath); err != nil {
		t.Fatal(err)
	}
	if err := service.SaveFileContent(tempDir, trackedPath, []byte("# Test Repository\nstaged\nunstaged")); err != nil {
		t.Fatal(err)
	}

	untrackedPath := "untracked.txt"
	if err := service.SaveFileContent(tempDir, untrackedPath, []byte("new file")); err != nil {
		t.Fatal(err)
	}

	status, err := service.GetRepositoryStatus(tempDir)
	if err != nil {
		t.Fatal(err)
	}

	if !containsString(status.Untracked, untrackedPath) {
		t.Errorf("Expected %s to be reported as untracked", untrackedPath)
	}
	if containsFileChangeStatus(status.Staged, "?") {
		t.Error("Untracked files should not appear in staged entries with status '?'")
	}
	if containsFileChangePath(status.Staged, untrackedPath) {
		t.Error("Untracked file should not appear in staged entries")
	}
	if !stagedStatusesOnlyAllowlisted(status.Staged) {
		t.Errorf("Expected staged statuses to be allowlisted, got %#v", status.Staged)
	}
	if !containsFileChangePath(status.Staged, trackedPath) {
		t.Errorf("Expected %s to appear in staged", trackedPath)
	}
	if !containsFileChangePath(status.Modified, trackedPath) {
		t.Errorf("Expected %s to appear in modified", trackedPath)
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

func TestBrowseDirectory_Root(t *testing.T) {
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
		"file2.txt",
		"dir1",
		"dir2",
	}

	for _, name := range testFiles {
		fullPath := filepath.Join(tempDir, name)
		if strings.HasSuffix(name, ".txt") {
			err = os.WriteFile(fullPath, []byte("test"), 0644)
		} else {
			err = os.MkdirAll(fullPath, 0755)
		}
		if err != nil {
			t.Fatal(err)
		}
	}

	// Browse root directory
	listing, err := service.BrowseDirectory(tempDir, "", 0, 100)
	if err != nil {
		t.Fatal(err)
	}

	if listing == nil {
		t.Fatal("BrowseDirectory returned nil")
	}

	// Should have 4 entries (2 files + 2 dirs)
	if len(listing.Entries) != 4 {
		t.Errorf("Expected 4 entries, got %d", len(listing.Entries))
	}

	// Check path is empty for root
	if listing.Path != "" {
		t.Errorf("Expected empty path for root, got %s", listing.Path)
	}

	// Check parent path is empty for root
	if listing.ParentPath != "" {
		t.Errorf("Expected empty parent_path for root, got %s", listing.ParentPath)
	}

	// Directories should come first
	if !listing.Entries[0].IsDirectory || !listing.Entries[1].IsDirectory {
		t.Error("Directories should be listed first")
	}
}

func TestBrowseDirectory_Subdirectory(t *testing.T) {
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

	// Create nested structure
	err = os.MkdirAll(filepath.Join(tempDir, "src", "components"), 0755)
	if err != nil {
		t.Fatal(err)
	}
	err = os.WriteFile(filepath.Join(tempDir, "src", "components", "Button.tsx"), []byte("btn"), 0644)
	if err != nil {
		t.Fatal(err)
	}
	err = os.WriteFile(filepath.Join(tempDir, "src", "index.ts"), []byte("idx"), 0644)
	if err != nil {
		t.Fatal(err)
	}

	// Browse src directory
	listing, err := service.BrowseDirectory(tempDir, "src", 0, 100)
	if err != nil {
		t.Fatal(err)
	}

	// Should have components dir and index.ts file
	if len(listing.Entries) != 2 {
		t.Errorf("Expected 2 entries in src, got %d", len(listing.Entries))
	}

	// Check path
	if listing.Path != "src" {
		t.Errorf("Expected path 'src', got %s", listing.Path)
	}

	// Check parent path
	if listing.ParentPath != "" {
		t.Errorf("Expected empty parent_path, got %s", listing.ParentPath)
	}

	// Browse src/components directory
	listing, err = service.BrowseDirectory(tempDir, "src/components", 0, 100)
	if err != nil {
		t.Fatal(err)
	}

	if len(listing.Entries) != 1 {
		t.Errorf("Expected 1 entry in components, got %d", len(listing.Entries))
	}

	// Check parent path
	if listing.ParentPath != "src" {
		t.Errorf("Expected parent_path 'src', got %s", listing.ParentPath)
	}
}

func TestBrowseDirectory_Pagination(t *testing.T) {
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

	// Create 10 files
	for i := 0; i < 10; i++ {
		err = os.WriteFile(filepath.Join(tempDir, fmt.Sprintf("file%d.txt", i)), []byte("test"), 0644)
		if err != nil {
			t.Fatal(err)
		}
	}

	// Get first page of 5
	listing, err := service.BrowseDirectory(tempDir, "", 0, 5)
	if err != nil {
		t.Fatal(err)
	}

	if len(listing.Entries) != 5 {
		t.Errorf("Expected 5 entries, got %d", len(listing.Entries))
	}

	if listing.TotalCount != 10 {
		t.Errorf("Expected total_count 10, got %d", listing.TotalCount)
	}

	if !listing.HasMore {
		t.Error("Expected has_more to be true")
	}

	if listing.Offset != 0 {
		t.Errorf("Expected offset 0, got %d", listing.Offset)
	}

	// Get second page
	listing2, err := service.BrowseDirectory(tempDir, "", 5, 5)
	if err != nil {
		t.Fatal(err)
	}

	if len(listing2.Entries) != 5 {
		t.Errorf("Expected 5 entries, got %d", len(listing2.Entries))
	}

	if listing2.HasMore {
		t.Error("Expected has_more to be false")
	}

	// Verify different files
	if listing.Entries[0].Name == listing2.Entries[0].Name {
		t.Error("Pages should have different files")
	}
}

func TestBrowseDirectory_PathTraversal(t *testing.T) {
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

	// Try to escape with ..
	_, err = service.BrowseDirectory(tempDir, "../", 0, 100)
	if err == nil {
		t.Error("Expected error for path traversal")
	}
}

func TestGetCommitFileDiff(t *testing.T) {
	s := NewService()

	// This test will fail until we implement the method
	_, err := s.GetCommitFileDiff("/tmp/test-repo", "abc123", "test.go", 0, 50)
	if err == nil {
		t.Error("Expected error for non-existent repo, got nil")
	}
}

func TestGetCommitFileDiff_InsertLineDoesNotCreateSyntheticDeleteAddPairs(t *testing.T) {
	service := NewService()

	tempDir, err := os.MkdirTemp("", "git_commit_diff_test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tempDir)

	_, err = service.InitRepository(tempDir)
	if err != nil {
		t.Fatal(err)
	}

	filePath := "test.txt"
	initialContent := []byte("line1\nline2\nline3\n")
	if err := service.SaveFileContent(tempDir, filePath, initialContent); err != nil {
		t.Fatal(err)
	}
	if err := service.StageFile(tempDir, filePath); err != nil {
		t.Fatal(err)
	}
	if err := service.CreateCommit(tempDir, models.CommitRequest{
		Message: "initial",
		Files:   []string{filePath},
		Author: models.Author{
			Name:  "Test User",
			Email: "test@example.com",
		},
	}); err != nil {
		t.Fatal(err)
	}

	updatedContent := []byte("line1\ninserted\nline2\nline3\n")
	if err := service.SaveFileContent(tempDir, filePath, updatedContent); err != nil {
		t.Fatal(err)
	}
	if err := service.StageFile(tempDir, filePath); err != nil {
		t.Fatal(err)
	}
	if err := service.CreateCommit(tempDir, models.CommitRequest{
		Message: "insert line",
		Files:   []string{filePath},
		Author: models.Author{
			Name:  "Test User",
			Email: "test@example.com",
		},
	}); err != nil {
		t.Fatal(err)
	}

	history, err := service.GetCommitHistory(tempDir, 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(history) == 0 {
		t.Fatal("Expected non-empty commit history")
	}

	latestCommitHash := history[0].Hash
	tokenized, err := service.GetCommitFileDiff(tempDir, latestCommitHash, filePath, 0, 50)
	if err != nil {
		t.Fatalf("GetCommitFileDiff failed: %v", err)
	}

	if tokenized.Additions != 1 {
		t.Fatalf("Expected exactly 1 addition for inserted line, got %d", tokenized.Additions)
	}
	if tokenized.Deletions != 0 {
		t.Fatalf("Expected 0 deletions for pure insertion, got %d", tokenized.Deletions)
	}
}
