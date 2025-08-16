package git

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"time"

	"gitweb/server/internal/models"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	diff "github.com/go-git/go-git/v5/plumbing/format/diff"
	"github.com/go-git/go-git/v5/plumbing/object"
)

type Service struct {
	repoPath string
}

func NewService() *Service {
	return &Service{}
}

func (s *Service) OpenRepository(path string) (*git.Repository, error) {
	return git.PlainOpen(path)
}

func (s *Service) CloneRepository(url, path string) (*git.Repository, error) {
	return git.PlainClone(path, false, &git.CloneOptions{
		URL: url,
	})
}

func (s *Service) InitRepository(path string) (*git.Repository, error) {
	return git.PlainInit(path, false)
}

func (s *Service) GetRepositoryStatus(repoPath string) (*models.RepositoryStatus, error) {
	repo, err := s.OpenRepository(repoPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open repository: %w", err)
	}

	worktree, err := repo.Worktree()
	if err != nil {
		return nil, fmt.Errorf("failed to get worktree: %w", err)
	}

	status, err := worktree.Status()
	if err != nil {
		return nil, fmt.Errorf("failed to get status: %w", err)
	}

	head, err := repo.Head()
	if err != nil {
		return nil, fmt.Errorf("failed to get HEAD: %w", err)
	}

	currentBranch := ""
	if head.Name().IsBranch() {
		currentBranch = head.Name().Short()
	}

	var staged []models.FileChange
	var modified []models.FileChange
	var untracked []string

	for file, fileStatus := range status {
		change := models.FileChange{
			Path:   file,
			Status: string(fileStatus.Staging),
			Type:   "file",
		}

		if fileStatus.Staging != git.Unmodified {
			staged = append(staged, change)
		}

		if fileStatus.Worktree != git.Unmodified {
			if fileStatus.Worktree == git.Untracked {
				untracked = append(untracked, file)
			} else {
				change.Status = string(fileStatus.Worktree)
				modified = append(modified, change)
			}
		}
	}

	return &models.RepositoryStatus{
		Branch:    currentBranch,
		IsClean:   status.IsClean(),
		Staged:    staged,
		Modified:  modified,
		Untracked: untracked,
		Conflicts: []string{},
		Ahead:     0,
		Behind:    0,
	}, nil
}

func (s *Service) GetBranches(repoPath string) ([]models.Branch, error) {
	repo, err := s.OpenRepository(repoPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open repository: %w", err)
	}

	branches := []models.Branch{}

	head, err := repo.Head()
	currentBranch := ""
	if err == nil && head.Name().IsBranch() {
		currentBranch = head.Name().Short()
	}

	branchIter, err := repo.Branches()
	if err != nil {
		return nil, fmt.Errorf("failed to get branches: %w", err)
	}

	err = branchIter.ForEach(func(ref *plumbing.Reference) error {
		branchName := ref.Name().Short()
		isCurrent := branchName == currentBranch

		commit, err := repo.CommitObject(ref.Hash())
		var lastCommit *models.Commit
		if err == nil {
			lastCommit = &models.Commit{
				Hash:    commit.Hash.String(),
				Message: strings.TrimSpace(commit.Message),
				Author: models.Author{
					Name:  commit.Author.Name,
					Email: commit.Author.Email,
				},
				Date: commit.Author.When,
			}
		}

		branches = append(branches, models.Branch{
			Name:       branchName,
			IsCurrent:  isCurrent,
			IsRemote:   false,
			LastCommit: lastCommit,
		})

		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("failed to iterate branches: %w", err)
	}

	return branches, nil
}

func (s *Service) GetCommitHistory(repoPath string, limit int) ([]models.Commit, error) {
	repo, err := s.OpenRepository(repoPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open repository: %w", err)
	}

	head, err := repo.Head()
	if err != nil {
		return nil, fmt.Errorf("failed to get HEAD: %w", err)
	}

	commitIter, err := repo.Log(&git.LogOptions{
		From: head.Hash(),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get commit log: %w", err)
	}

	commits := []models.Commit{}
	count := 0

	err = commitIter.ForEach(func(commit *object.Commit) error {
		if limit > 0 && count >= limit {
			return nil
		}

		parentHash := ""
		if len(commit.ParentHashes) > 0 {
			parentHash = commit.ParentHashes[0].String()
		}

		commits = append(commits, models.Commit{
			Hash:    commit.Hash.String(),
			Message: strings.TrimSpace(commit.Message),
			Author: models.Author{
				Name:  commit.Author.Name,
				Email: commit.Author.Email,
			},
			Date:       commit.Author.When,
			ParentHash: parentHash,
		})

		count++
		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("failed to iterate commits: %w", err)
	}

	return commits, nil
}

func (s *Service) CreateCommit(repoPath string, req models.CommitRequest) error {
	repo, err := s.OpenRepository(repoPath)
	if err != nil {
		return fmt.Errorf("failed to open repository: %w", err)
	}

	worktree, err := repo.Worktree()
	if err != nil {
		return fmt.Errorf("failed to get worktree: %w", err)
	}

	for _, file := range req.Files {
		_, err = worktree.Add(file)
		if err != nil {
			return fmt.Errorf("failed to add file %s: %w", file, err)
		}
	}

	commitOptions := &git.CommitOptions{
		Author: &object.Signature{
			Name:  req.Author.Name,
			Email: req.Author.Email,
			When:  time.Now(),
		},
	}

	if req.Author.Name == "" {
		commitOptions.Author = nil
	}

	_, err = worktree.Commit(req.Message, commitOptions)
	if err != nil {
		return fmt.Errorf("failed to create commit: %w", err)
	}

	return nil
}

func (s *Service) CreateBranch(repoPath, branchName string) error {
	repo, err := s.OpenRepository(repoPath)
	if err != nil {
		return fmt.Errorf("failed to open repository: %w", err)
	}

	head, err := repo.Head()
	if err != nil {
		return fmt.Errorf("failed to get HEAD: %w", err)
	}

	branchRef := plumbing.NewBranchReferenceName(branchName)
	ref := plumbing.NewHashReference(branchRef, head.Hash())

	err = repo.Storer.SetReference(ref)
	if err != nil {
		return fmt.Errorf("failed to create branch: %w", err)
	}

	return nil
}

func (s *Service) SwitchBranch(repoPath, branchName string) error {
	repo, err := s.OpenRepository(repoPath)
	if err != nil {
		return fmt.Errorf("failed to open repository: %w", err)
	}

	worktree, err := repo.Worktree()
	if err != nil {
		return fmt.Errorf("failed to get worktree: %w", err)
	}

	err = worktree.Checkout(&git.CheckoutOptions{
		Branch: plumbing.NewBranchReferenceName(branchName),
	})
	if err != nil {
		return fmt.Errorf("failed to switch branch: %w", err)
	}

	return nil
}

func (s *Service) GetFileContent(repoPath, filePath string) ([]byte, error) {
	fullPath := filepath.Join(repoPath, filePath)
	return os.ReadFile(fullPath)
}

func (s *Service) SaveFileContent(repoPath, filePath string, content []byte) error {
	fullPath := filepath.Join(repoPath, filePath)

	dir := filepath.Dir(fullPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create directories: %w", err)
	}

	return os.WriteFile(fullPath, content, 0644)
}

func (s *Service) GetFileTree(repoPath string) ([]models.FileInfo, error) {
	var files []models.FileInfo

	// Initialize gitignore parser
	gitignore := NewGitIgnore(repoPath)

	err := filepath.WalkDir(repoPath, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		if strings.Contains(path, ".git") && d.IsDir() {
			return fs.SkipDir
		}

		if path == repoPath {
			return nil
		}

		relativePath, err := filepath.Rel(repoPath, path)
		if err != nil {
			return err
		}

		// Check if the file/directory should be ignored
		if gitignore.IsIgnored(relativePath, d.IsDir()) {
			if d.IsDir() {
				return fs.SkipDir
			}
			return nil
		}

		info, err := d.Info()
		if err != nil {
			return err
		}

		files = append(files, models.FileInfo{
			Path:        relativePath,
			Name:        d.Name(),
			IsDirectory: d.IsDir(),
			Size:        info.Size(),
			ModTime:     info.ModTime(),
			Mode:        info.Mode().String(),
		})

		return nil
	})

	return files, err
}

func (s *Service) Push(repoPath string) error {
	repo, err := s.OpenRepository(repoPath)
	if err != nil {
		return fmt.Errorf("failed to open repository: %w", err)
	}

	err = repo.Push(&git.PushOptions{})
	if err != nil {
		return fmt.Errorf("failed to push: %w", err)
	}

	return nil
}

func (s *Service) Pull(repoPath string) error {
	repo, err := s.OpenRepository(repoPath)
	if err != nil {
		return fmt.Errorf("failed to open repository: %w", err)
	}

	worktree, err := repo.Worktree()
	if err != nil {
		return fmt.Errorf("failed to get worktree: %w", err)
	}

	err = worktree.Pull(&git.PullOptions{
		RemoteName: "origin",
	})
	if err != nil && err != git.NoErrAlreadyUpToDate {
		return fmt.Errorf("failed to pull: %w", err)
	}

	return nil
}

func (s *Service) StageFile(repoPath, filePath string) error {
	repo, err := s.OpenRepository(repoPath)
	if err != nil {
		return fmt.Errorf("failed to open repository: %w", err)
	}

	worktree, err := repo.Worktree()
	if err != nil {
		return fmt.Errorf("failed to get worktree: %w", err)
	}

	_, err = worktree.Add(filePath)
	if err != nil {
		return fmt.Errorf("failed to stage file: %w", err)
	}

	return nil
}

func (s *Service) UnstageFile(repoPath, filePath string) error {
	repo, err := s.OpenRepository(repoPath)
	if err != nil {
		return fmt.Errorf("failed to open repository: %w", err)
	}

	worktree, err := repo.Worktree()
	if err != nil {
		return fmt.Errorf("failed to get worktree: %w", err)
	}

	err = worktree.Reset(&git.ResetOptions{
		Mode: git.MixedReset,
	})
	if err != nil {
		return fmt.Errorf("failed to unstage file: %w", err)
	}

	// Re-add all files except the one we want to unstage
	status, err := worktree.Status()
	if err != nil {
		return fmt.Errorf("failed to get status: %w", err)
	}

	for file, fileStatus := range status {
		if file != filePath && fileStatus.Staging != git.Unmodified {
			_, err = worktree.Add(file)
			if err != nil {
				return fmt.Errorf("failed to re-stage file %s: %w", file, err)
			}
		}
	}

	return nil
}

func (s *Service) GetCommitDetails(repoPath, commitHash string) (*models.CommitDetail, error) {
	repo, err := s.OpenRepository(repoPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open repository: %w", err)
	}

	hash := plumbing.NewHash(commitHash)
	commit, err := repo.CommitObject(hash)
	if err != nil {
		return nil, fmt.Errorf("failed to get commit: %w", err)
	}

	// Get parent commit if exists
	var parentCommit *object.Commit
	if len(commit.ParentHashes) > 0 {
		parentCommit, _ = repo.CommitObject(commit.ParentHashes[0])
	}

	// Get the file changes (diff)
	var changes []models.FileDiff
	var stats models.DiffStats

	if parentCommit != nil {
		// Compare with parent
		parentTree, err := parentCommit.Tree()
		if err != nil {
			return nil, fmt.Errorf("failed to get parent tree: %w", err)
		}

		commitTree, err := commit.Tree()
		if err != nil {
			return nil, fmt.Errorf("failed to get commit tree: %w", err)
		}

		patch, err := parentTree.Patch(commitTree)
		if err != nil {
			return nil, fmt.Errorf("failed to get patch: %w", err)
		}

		for _, filePatch := range patch.FilePatches() {
			from, to := filePatch.Files()

			var filePath string
			var changeType string

			if from == nil && to != nil {
				// New file
				filePath = to.Path()
				changeType = "added"
			} else if from != nil && to == nil {
				// Deleted file
				filePath = from.Path()
				changeType = "deleted"
			} else if from != nil && to != nil {
				// Modified file
				filePath = to.Path()
				changeType = "modified"
			}

			// Build patch string containing only changed lines and
			// count additions/deletions
			var patchBuilder strings.Builder
			additions := 0
			deletions := 0
			for _, chunk := range filePatch.Chunks() {
				switch chunk.Type() {
				case diff.Add:
					lines := strings.Split(chunk.Content(), "\n")
					for _, line := range lines {
						if line == "" {
							continue
						}
						patchBuilder.WriteString("+" + line + "\n")
						additions++
					}
				case diff.Delete:
					lines := strings.Split(chunk.Content(), "\n")
					for _, line := range lines {
						if line == "" {
							continue
						}
						patchBuilder.WriteString("-" + line + "\n")
						deletions++
					}
				}
			}

			patchContent := strings.TrimSuffix(patchBuilder.String(), "\n")
			stats.Additions += additions
			stats.Deletions += deletions

			changes = append(changes, models.FileDiff{
				Path:       filePath,
				ChangeType: changeType,
				Additions:  additions,
				Deletions:  deletions,
				Patch:      patchContent,
			})
		}
	} else {
		// Initial commit - all files are additions
		tree, err := commit.Tree()
		if err != nil {
			return nil, fmt.Errorf("failed to get tree: %w", err)
		}

		err = tree.Files().ForEach(func(file *object.File) error {
			changes = append(changes, models.FileDiff{
				Path:       file.Name,
				ChangeType: "added",
				Additions:  0,
				Deletions:  0,
				Patch:      "",
			})
			stats.Additions++
			return nil
		})
		if err != nil {
			return nil, fmt.Errorf("failed to iterate files: %w", err)
		}
	}

	parentHash := ""
	if len(commit.ParentHashes) > 0 {
		parentHash = commit.ParentHashes[0].String()
	}

	return &models.CommitDetail{
		Hash:    commit.Hash.String(),
		Message: strings.TrimSpace(commit.Message),
		Author: models.Author{
			Name:  commit.Author.Name,
			Email: commit.Author.Email,
		},
		Date:       commit.Author.When,
		ParentHash: parentHash,
		Changes:    changes,
		Stats:      stats,
	}, nil
}

func (s *Service) DeleteBranch(repoPath, branchName string) error {
	repo, err := s.OpenRepository(repoPath)
	if err != nil {
		return fmt.Errorf("failed to open repository: %w", err)
	}

	// Check if it's the current branch
	head, err := repo.Head()
	if err != nil {
		return fmt.Errorf("failed to get HEAD: %w", err)
	}

	currentBranch := ""
	if head.Name().IsBranch() {
		currentBranch = head.Name().Short()
	}

	if currentBranch == branchName {
		return fmt.Errorf("cannot delete current branch: %s", branchName)
	}

	// Delete the branch reference
	branchRef := plumbing.NewBranchReferenceName(branchName)
	err = repo.Storer.RemoveReference(branchRef)
	if err != nil {
		return fmt.Errorf("failed to delete branch: %w", err)
	}

	return nil
}

func (s *Service) GetFileDiff(repoPath, filePath string) (string, error) {
	repo, err := s.OpenRepository(repoPath)
	if err != nil {
		return "", fmt.Errorf("failed to open repository: %w", err)
	}

	worktree, err := repo.Worktree()
	if err != nil {
		return "", fmt.Errorf("failed to get worktree: %w", err)
	}

	// Get the current status
	status, err := worktree.Status()
	if err != nil {
		return "", fmt.Errorf("failed to get status: %w", err)
	}

	fileStatus, exists := status[filePath]
	if !exists {
		return "", fmt.Errorf("file not found in status: %s", filePath)
	}

	// Handle different file statuses
	switch fileStatus.Worktree {
	case git.Untracked:
		// For untracked files, show the entire content as additions
		content, err := os.ReadFile(filepath.Join(repoPath, filePath))
		if err != nil {
			return "", fmt.Errorf("failed to read file: %w", err)
		}

		var diff strings.Builder
		diff.WriteString(fmt.Sprintf("diff --git a/%s b/%s\n", filePath, filePath))
		diff.WriteString("new file mode 100644\n")
		diff.WriteString("index 0000000..0000000\n")
		diff.WriteString("--- /dev/null\n")
		diff.WriteString(fmt.Sprintf("+++ b/%s\n", filePath))

		lines := strings.Split(string(content), "\n")
		if len(lines) > 0 {
			diff.WriteString(fmt.Sprintf("@@ -0,0 +1,%d @@\n", len(lines)))
			for _, line := range lines {
				diff.WriteString("+" + line + "\n")
			}
		}

		return diff.String(), nil

	case git.Modified, git.Added:
		// Get HEAD commit
		head, err := repo.Head()
		if err != nil {
			return "", fmt.Errorf("failed to get HEAD: %w", err)
		}

		commit, err := repo.CommitObject(head.Hash())
		if err != nil {
			return "", fmt.Errorf("failed to get commit: %w", err)
		}

		// Get the file from HEAD
		tree, err := commit.Tree()
		if err != nil {
			return "", fmt.Errorf("failed to get tree: %w", err)
		}

		var oldContent string
		file, err := tree.File(filePath)
		if err != nil {
			// File doesn't exist in HEAD (new file)
			oldContent = ""
		} else {
			oldContent, err = file.Contents()
			if err != nil {
				return "", fmt.Errorf("failed to get file contents: %w", err)
			}
		}

		// Get current file content
		newContent, err := os.ReadFile(filepath.Join(repoPath, filePath))
		if err != nil {
			return "", fmt.Errorf("failed to read current file: %w", err)
		}

		// Generate diff
		return s.generateTextDiff(filePath, oldContent, string(newContent)), nil

	case git.Deleted:
		// Get the file content from HEAD
		head, err := repo.Head()
		if err != nil {
			return "", fmt.Errorf("failed to get HEAD: %w", err)
		}

		commit, err := repo.CommitObject(head.Hash())
		if err != nil {
			return "", fmt.Errorf("failed to get commit: %w", err)
		}

		tree, err := commit.Tree()
		if err != nil {
			return "", fmt.Errorf("failed to get tree: %w", err)
		}

		file, err := tree.File(filePath)
		if err != nil {
			return "", fmt.Errorf("failed to get deleted file: %w", err)
		}

		oldContent, err := file.Contents()
		if err != nil {
			return "", fmt.Errorf("failed to get file contents: %w", err)
		}

		return s.generateTextDiff(filePath, oldContent, ""), nil

	default:
		return "", fmt.Errorf("unsupported file status: %v", fileStatus.Worktree)
	}
}

// generateTextDiff creates a unified diff between two text contents
func (s *Service) generateTextDiff(filePath, oldContent, newContent string) string {
	var diff strings.Builder

	diff.WriteString(fmt.Sprintf("diff --git a/%s b/%s\n", filePath, filePath))
	diff.WriteString("index 0000000..0000000 100644\n")
	diff.WriteString(fmt.Sprintf("--- a/%s\n", filePath))
	diff.WriteString(fmt.Sprintf("+++ b/%s\n", filePath))

	oldLines := strings.Split(oldContent, "\n")
	newLines := strings.Split(newContent, "\n")

	// Simple line-by-line diff (this could be improved with a proper diff algorithm)
	maxLines := max(len(oldLines), len(newLines))

	if maxLines > 0 {
		diff.WriteString(fmt.Sprintf("@@ -1,%d +1,%d @@\n", len(oldLines), len(newLines)))

		for i := 0; i < maxLines; i++ {
			var oldLine, newLine string

			if i < len(oldLines) {
				oldLine = oldLines[i]
			}
			if i < len(newLines) {
				newLine = newLines[i]
			}

			if i < len(oldLines) && i < len(newLines) {
				if oldLine != newLine {
					// Changed line
					diff.WriteString("-" + oldLine + "\n")
					diff.WriteString("+" + newLine + "\n")
				} else {
					// Unchanged line (context)
					diff.WriteString(" " + oldLine + "\n")
				}
			} else if i < len(oldLines) {
				// Deleted line
				diff.WriteString("-" + oldLine + "\n")
			} else if i < len(newLines) {
				// Added line
				diff.WriteString("+" + newLine + "\n")
			}
		}
	}

	return diff.String()
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
