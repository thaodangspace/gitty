package git

import (
	"bufio"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// GitIgnore represents a .gitignore parser
type GitIgnore struct {
	patterns []gitignorePattern
}

type gitignorePattern struct {
	pattern *regexp.Regexp
	negate  bool
	isDir   bool
}

// NewGitIgnore creates a new GitIgnore parser for the given repository path
func NewGitIgnore(repoPath string) *GitIgnore {
	gi := &GitIgnore{
		patterns: make([]gitignorePattern, 0),
	}
	
	// Load .gitignore from the repository root
	gitignorePath := filepath.Join(repoPath, ".gitignore")
	gi.loadGitIgnoreFile(gitignorePath)
	
	return gi
}

// loadGitIgnoreFile loads patterns from a .gitignore file
func (gi *GitIgnore) loadGitIgnoreFile(path string) {
	file, err := os.Open(path)
	if err != nil {
		// .gitignore file doesn't exist, which is fine
		return
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		
		// Skip empty lines and comments
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		
		gi.addPattern(line)
	}
}

// addPattern adds a gitignore pattern
func (gi *GitIgnore) addPattern(pattern string) {
	negate := false
	isDir := false
	
	// Handle negation
	if strings.HasPrefix(pattern, "!") {
		negate = true
		pattern = pattern[1:]
	}
	
	// Handle directory patterns
	if strings.HasSuffix(pattern, "/") {
		isDir = true
		pattern = strings.TrimSuffix(pattern, "/")
	}
	
	// Convert gitignore pattern to regex
	regexPattern := gitignorePatternToRegex(pattern)
	regex, err := regexp.Compile(regexPattern)
	if err != nil {
		// Skip invalid patterns
		return
	}
	
	gi.patterns = append(gi.patterns, gitignorePattern{
		pattern: regex,
		negate:  negate,
		isDir:   isDir,
	})
}

// gitignorePatternToRegex converts a gitignore pattern to a regex pattern
func gitignorePatternToRegex(pattern string) string {
	// Handle leading slash (absolute path from repo root)
	absolute := strings.HasPrefix(pattern, "/")
	if absolute {
		pattern = pattern[1:]
	}
	
	// Escape regex special characters, but preserve our placeholders
	result := strings.ReplaceAll(pattern, "**", "DOUBLESTAR")
	result = strings.ReplaceAll(result, "*", "STAR")
	result = strings.ReplaceAll(result, "?", "QUESTION")
	result = regexp.QuoteMeta(result)
	
	// Replace placeholders with regex equivalents
	result = strings.ReplaceAll(result, "DOUBLESTAR", ".*")  // ** matches any number of directories and files
	result = strings.ReplaceAll(result, "STAR", "[^/]*")     // * matches anything except /
	result = strings.ReplaceAll(result, "QUESTION", "[^/]")  // ? matches any single character except /
	
	if absolute {
		// Absolute path from repo root
		result = "^" + result + "(/.*)?$"
	} else {
		// Pattern can match at any directory level
		result = "(^|.*?/)" + result + "(/.*)?$"
	}
	
	return result
}

// IsIgnored checks if a given path should be ignored
func (gi *GitIgnore) IsIgnored(path string, isDir bool) bool {
	// Normalize path separators
	path = filepath.ToSlash(path)
	
	ignored := false
	
	for _, p := range gi.patterns {
		// Skip directory-only patterns for files
		if p.isDir && !isDir {
			continue
		}
		
		if p.pattern.MatchString(path) {
			if p.negate {
				ignored = false
			} else {
				ignored = true
			}
		}
	}
	
	return ignored
}