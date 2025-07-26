package filesystem

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"time"

	"gitweb/server/internal/models"
)

type Service struct {
	restrictToUserHome bool
	userHomeDir        string
}

func NewService(restrictToUserHome bool) *Service {
	homeDir, _ := os.UserHomeDir()
	return &Service{
		restrictToUserHome: restrictToUserHome,
		userHomeDir:        homeDir,
	}
}

func (s *Service) BrowseDirectory(path string) (*models.DirectoryListing, error) {
	if path == "" {
		path = s.getDefaultStartPath()
	}

	cleanPath := filepath.Clean(path)

	if !s.isPathAllowed(cleanPath) {
		return nil, fmt.Errorf("access denied: path outside allowed directory")
	}

	stat, err := os.Stat(cleanPath)
	if err != nil {
		return nil, fmt.Errorf("failed to access path: %w", err)
	}

	if !stat.IsDir() {
		return nil, fmt.Errorf("path is not a directory")
	}

	entries, err := os.ReadDir(cleanPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read directory: %w", err)
	}

	var directoryEntries []models.DirectoryEntry

	for _, entry := range entries {
		info, err := entry.Info()
		if err != nil {
			continue
		}

		entryPath := filepath.Join(cleanPath, entry.Name())

		dirEntry := models.DirectoryEntry{
			Name:        entry.Name(),
			Path:        entryPath,
			IsDirectory: entry.IsDir(),
			IsHidden:    s.isHidden(entry.Name()),
			Size:        info.Size(),
			ModTime:     info.ModTime().Format(time.RFC3339),
			Permissions: info.Mode().String(),
			IsGitRepo:   s.isGitRepository(entryPath),
		}

		directoryEntries = append(directoryEntries, dirEntry)
	}

	sort.Slice(directoryEntries, func(i, j int) bool {
		if directoryEntries[i].IsDirectory != directoryEntries[j].IsDirectory {
			return directoryEntries[i].IsDirectory
		}
		return directoryEntries[i].Name < directoryEntries[j].Name
	})

	parentPath := filepath.Dir(cleanPath)
	canGoUp := s.canGoUp(cleanPath, parentPath)

	if !canGoUp {
		parentPath = ""
	}

	return &models.DirectoryListing{
		CurrentPath: cleanPath,
		ParentPath:  parentPath,
		Entries:     directoryEntries,
		CanGoUp:     canGoUp,
	}, nil
}

func (s *Service) getDefaultStartPath() string {
	if s.userHomeDir != "" {
		return s.userHomeDir
	}

	if runtime.GOOS == "windows" {
		return "C:\\"
	}
	return "/"
}

func (s *Service) isPathAllowed(path string) bool {
	if !s.restrictToUserHome {
		return true
	}

	if s.userHomeDir == "" {
		return false
	}

	absPath, err := filepath.Abs(path)
	if err != nil {
		return false
	}

	absHome, err := filepath.Abs(s.userHomeDir)
	if err != nil {
		return false
	}

	rel, err := filepath.Rel(absHome, absPath)
	if err != nil {
		return false
	}

	return !strings.HasPrefix(rel, "..") && rel != ".."
}

func (s *Service) canGoUp(currentPath, parentPath string) bool {
	if currentPath == parentPath {
		return false
	}

	if !s.isPathAllowed(parentPath) {
		return false
	}

	if runtime.GOOS == "windows" {
		return len(currentPath) > 3
	}

	return currentPath != "/"
}

func (s *Service) isHidden(name string) bool {
	if runtime.GOOS == "windows" {
		return false
	}
	return strings.HasPrefix(name, ".")
}

func (s *Service) isGitRepository(path string) bool {
	gitPath := filepath.Join(path, ".git")
	stat, err := os.Stat(gitPath)
	if err != nil {
		return false
	}
	return stat.IsDir()
}

func (s *Service) GetVolumeRoots() ([]models.DirectoryEntry, error) {
	var roots []models.DirectoryEntry

	if runtime.GOOS == "windows" {
		drives := []string{"C:\\", "D:\\", "E:\\", "F:\\", "G:\\", "H:\\"}
		for _, drive := range drives {
			if _, err := os.Stat(drive); err == nil {
				roots = append(roots, models.DirectoryEntry{
					Name:        drive,
					Path:        drive,
					IsDirectory: true,
					IsHidden:    false,
					IsGitRepo:   false,
				})
			}
		}
	} else {
		roots = append(roots, models.DirectoryEntry{
			Name:        "/",
			Path:        "/",
			IsDirectory: true,
			IsHidden:    false,
			IsGitRepo:   false,
		})

		if s.userHomeDir != "" {
			roots = append(roots, models.DirectoryEntry{
				Name:        "Home",
				Path:        s.userHomeDir,
				IsDirectory: true,
				IsHidden:    false,
				IsGitRepo:   false,
			})
		}
	}

	return roots, nil
}