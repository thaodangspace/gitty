package git

import (
	"os"
	"path/filepath"
	"testing"
)

func TestGitIgnore(t *testing.T) {
	// Create a temporary directory for testing
	tempDir, err := os.MkdirTemp("", "gitignore_test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tempDir)

	// Create a test .gitignore file
	gitignoreContent := `# Test gitignore
node_modules/
*.log
temp/
!important.log
build
*.tmp
`
	gitignorePath := filepath.Join(tempDir, ".gitignore")
	err = os.WriteFile(gitignorePath, []byte(gitignoreContent), 0644)
	if err != nil {
		t.Fatal(err)
	}

	// Initialize GitIgnore
	gi := NewGitIgnore(tempDir)

	// Test cases
	testCases := []struct {
		path     string
		isDir    bool
		expected bool
	}{
		{"node_modules", true, true},          // Directory should be ignored
		{"node_modules/package.json", false, true}, // File in ignored directory
		{"app.log", false, true},              // File matching *.log pattern
		{"important.log", false, false},       // Negated pattern
		{"src/app.js", false, false},          // Regular file
		{"temp", true, true},                  // Directory matching temp/
		{"temp/file.txt", false, true},        // File in ignored directory
		{"build", false, true},                // File matching build pattern
		{"build", true, true},                 // Directory matching build pattern  
		{"file.tmp", false, true},             // File matching *.tmp pattern
		{"src/file.txt", false, false},        // Regular file in subdirectory
	}

	for _, tc := range testCases {
		result := gi.IsIgnored(tc.path, tc.isDir)
		if result != tc.expected {
			t.Errorf("IsIgnored(%q, %v) = %v, expected %v", tc.path, tc.isDir, result, tc.expected)
		}
	}
}

func TestGitIgnorePatternToRegex(t *testing.T) {
	testCases := []struct {
		pattern  string
		expected string
	}{
		{"*.log", "(^|.*?/)[^/]*\\.log(/.*)?$"},
		{"node_modules", "(^|.*?/)node_modules(/.*)?$"},
		{"/build", "^build(/.*)?$"},
	}

	for _, tc := range testCases {
		result := gitignorePatternToRegex(tc.pattern)
		if result != tc.expected {
			t.Errorf("gitignorePatternToRegex(%q) = %q, expected %q", tc.pattern, result, tc.expected)
		}
	}
}