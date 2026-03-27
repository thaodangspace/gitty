package git

import (
	"fmt"
	"io"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
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

func isAllowlistedStagedStatus(status git.StatusCode) bool {
	switch status {
	case git.Added, git.Modified, git.Deleted, git.Renamed, git.Copied:
		return true
	default:
		return false
	}
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

	// Initialize gitignore parser to filter out ignored files
	gitignore := NewGitIgnore(repoPath)

	for file, fileStatus := range status {
		change := models.FileChange{
			Path:   file,
			Status: string(fileStatus.Staging),
			Type:   "file",
		}

		if isAllowlistedStagedStatus(fileStatus.Staging) {
			staged = append(staged, change)
		}

		if fileStatus.Worktree != git.Unmodified {
			if fileStatus.Worktree == git.Untracked {
				// Skip untracked files that are gitignored
				if !gitignore.IsIgnored(file, false) {
					untracked = append(untracked, file)
				}
			} else {
				change.Status = string(fileStatus.Worktree)
				modified = append(modified, change)
			}
		}
	}

	// Calculate ahead/behind counts
	ahead, behind := 0, 0
	if head.Name().IsBranch() {
		// Get the remote tracking branch
		remoteBranchName := plumbing.NewRemoteReferenceName("origin", currentBranch)
		remoteBranchRef, err := repo.Reference(remoteBranchName, true)

		if err == nil {
			// Remote branch exists, calculate ahead/behind
			localCommit, err := repo.CommitObject(head.Hash())
			if err == nil {
				remoteCommit, err := repo.CommitObject(remoteBranchRef.Hash())
				if err == nil {
					// Count commits ahead (local commits not in remote)
					commitIter, err := repo.Log(&git.LogOptions{
						From: head.Hash(),
					})
					if err == nil {
						err = commitIter.ForEach(func(commit *object.Commit) error {
							if commit.Hash == remoteCommit.Hash {
								return fmt.Errorf("found common ancestor")
							}
							ahead++
							return nil
						})
						// Reset ahead if we didn't find the remote commit (diverged branches)
						if err != nil && err.Error() != "found common ancestor" {
							ahead = 0
						}
					}

					// Count commits behind (remote commits not in local)
					commitIter, err = repo.Log(&git.LogOptions{
						From: remoteBranchRef.Hash(),
					})
					if err == nil {
						err = commitIter.ForEach(func(commit *object.Commit) error {
							if commit.Hash == localCommit.Hash {
								return fmt.Errorf("found common ancestor")
							}
							behind++
							return nil
						})
						// Reset behind if we didn't find the local commit (diverged branches)
						if err != nil && err.Error() != "found common ancestor" {
							behind = 0
						}
					}
				}
			}
		}
	}

	// Sort files alphabetically for consistent ordering
	sort.Slice(staged, func(i, j int) bool {
		return staged[i].Path < staged[j].Path
	})
	sort.Slice(modified, func(i, j int) bool {
		return modified[i].Path < modified[j].Path
	})
	sort.Strings(untracked)

	return &models.RepositoryStatus{
		Branch:    currentBranch,
		IsClean:   status.IsClean(),
		Staged:    staged,
		Modified:  modified,
		Untracked: untracked,
		Conflicts: []string{},
		Ahead:     ahead,
		Behind:    behind,
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

// BrowseDirectory reads a single directory level with pagination
// Unlike GetFileTree which recursively walks the entire tree, this only reads the immediate children
func (s *Service) BrowseDirectory(repoPath, subPath string, offset, limit int) (*models.RepoDirectoryListing, error) {
	// Resolve target path
	targetPath := filepath.Join(repoPath, subPath)

	// Validate path is within repo (security check)
	cleanTarget := filepath.Clean(targetPath)
	cleanRepo := filepath.Clean(repoPath)
	if !strings.HasPrefix(cleanTarget, cleanRepo) {
		return nil, fmt.Errorf("path outside repository")
	}

	// Read directory entries (non-recursive)
	entries, err := os.ReadDir(targetPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read directory: %w", err)
	}

	// Initialize gitignore matcher
	gitignore := NewGitIgnore(repoPath)

	// Process entries
	var fileInfos []models.FileInfo
	for _, entry := range entries {
		// SECURITY: Skip symlinks to prevent path traversal attacks
		if entry.Type() == os.ModeSymlink {
			continue
		}

		relativePath := filepath.Join(subPath, entry.Name())

		// Skip .git directory
		if entry.Name() == ".git" {
			continue
		}

		// Skip gitignored entries
		if gitignore.IsIgnored(relativePath, entry.IsDir()) {
			continue
		}

		info, err := entry.Info()
		if err != nil {
			continue
		}

		fileInfos = append(fileInfos, models.FileInfo{
			Path:        relativePath,
			Name:        entry.Name(),
			IsDirectory: entry.IsDir(),
			Size:        info.Size(),
			ModTime:     info.ModTime(),
			Mode:        info.Mode().String(),
		})
	}

	// Sort: directories first, then alphabetically
	sort.Slice(fileInfos, func(i, j int) bool {
		if fileInfos[i].IsDirectory != fileInfos[j].IsDirectory {
			return fileInfos[i].IsDirectory
		}
		return fileInfos[i].Name < fileInfos[j].Name
	})

	// Apply pagination
	totalCount := len(fileInfos)
	hasMore := offset+limit < totalCount

	if offset > totalCount {
		offset = totalCount
	}
	end := offset + limit
	if end > totalCount {
		end = totalCount
	}

	paginatedEntries := fileInfos[offset:end]

	// Determine parent path
	parentPath := ""
	if subPath != "" {
		parentPath = filepath.Dir(subPath)
		if parentPath == "." {
			parentPath = ""
		}
	}

	return &models.RepoDirectoryListing{
		Path:       subPath,
		ParentPath: parentPath,
		Entries:    paginatedEntries,
		TotalCount: totalCount,
		HasMore:    hasMore,
		Offset:     offset,
	}, nil
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

func (s *Service) ForcePush(repoPath string) error {
	repo, err := s.OpenRepository(repoPath)
	if err != nil {
		return fmt.Errorf("failed to open repository: %w", err)
	}

	err = repo.Push(&git.PushOptions{
		Force: true,
	})
	if err != nil {
		return fmt.Errorf("failed to force push: %w", err)
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

func (s *Service) StageAll(repoPath string) error {
	repo, err := s.OpenRepository(repoPath)
	if err != nil {
		return fmt.Errorf("failed to open repository: %w", err)
	}

	worktree, err := repo.Worktree()
	if err != nil {
		return fmt.Errorf("failed to get worktree: %w", err)
	}

	_, err = worktree.Add(".")
	if err != nil {
		return fmt.Errorf("failed to stage all files: %w", err)
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

	err = worktree.Restore(&git.RestoreOptions{
		Staged: true,
		Files:  []string{filePath},
	})
	if err != nil {
		return fmt.Errorf("failed to unstage file: %w", err)
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
				content := chunk.Content()
				if content == "" {
					continue
				}
				lines := strings.Split(content, "\n")
				switch chunk.Type() {
				case diff.Add:
					for _, line := range lines {
						if line == "" {
							continue
						}
						if !strings.HasPrefix(line, "+") {
							line = "+" + line
						}
						patchBuilder.WriteString(line + "\n")
						additions++
					}
				case diff.Delete:
					for _, line := range lines {
						if line == "" {
							continue
						}
						if !strings.HasPrefix(line, "-") {
							line = "-" + line
						}
						patchBuilder.WriteString(line + "\n")
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

	case git.Unmodified:
		return "", nil

	default:
		return "", fmt.Errorf("unsupported file status: %v", fileStatus.Worktree)
	}
}

// GetStagedDiff gets the diff of a staged file (index vs HEAD)
func (s *Service) GetStagedDiff(repoPath, filePath string) (string, error) {
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

	// Check the staging status
	if fileStatus.Staging == git.Unmodified {
		return "", fmt.Errorf("file not staged: %s", filePath)
	}

	// Get HEAD commit
	head, err := repo.Head()
	if err != nil {
		return "", fmt.Errorf("failed to get HEAD: %w", err)
	}

	var oldContent string
	var oldFileExists bool

	if head.Hash().IsZero() {
		// No HEAD commit yet, empty repo
		oldContent = ""
		oldFileExists = false
	} else {
		commit, err := repo.CommitObject(head.Hash())
		if err != nil {
			return "", fmt.Errorf("failed to get commit: %w", err)
		}

		// Get the file from HEAD
		tree, err := commit.Tree()
		if err != nil {
			return "", fmt.Errorf("failed to get tree: %w", err)
		}

		file, err := tree.File(filePath)
		if err != nil {
			// File doesn't exist in HEAD (new file)
			oldContent = ""
			oldFileExists = false
		} else {
			oldContent, err = file.Contents()
			if err != nil {
				return "", fmt.Errorf("failed to get file contents: %w", err)
			}
			oldFileExists = true
		}
	}

	// Get the staged content from the index
	index, err := repo.Storer.Index()
	if err != nil {
		return "", fmt.Errorf("failed to get index: %w", err)
	}

	entry, err := index.Entry(filePath)
	if err != nil {
		return "", fmt.Errorf("failed to get index entry: %w", err)
	}

	// Read the staged content from the object database
	obj, err := repo.Storer.EncodedObject(plumbing.BlobObject, entry.Hash)
	if err != nil {
		return "", fmt.Errorf("failed to get staged object: %w", err)
	}

	reader, err := obj.Reader()
	if err != nil {
		return "", fmt.Errorf("failed to get object reader: %w", err)
	}
	defer reader.Close()

	stagedContent, err := io.ReadAll(reader)
	if err != nil {
		return "", fmt.Errorf("failed to read staged content: %w", err)
	}

	// Generate diff
	if fileStatus.Staging == git.Deleted {
		// File was staged for deletion
		return s.generateTextDiff(filePath, oldContent, ""), nil
	}

	// For new or modified files
	return s.generateStagedDiff(filePath, oldContent, oldFileExists, string(stagedContent)), nil
}

// generateStagedDiff creates a unified diff for staged changes
func (s *Service) generateStagedDiff(filePath, oldContent string, oldFileExists bool, newContent string) string {
	var diff strings.Builder

	diff.WriteString(fmt.Sprintf("diff --git a/%s b/%s\n", filePath, filePath))

	if !oldFileExists {
		// New file
		diff.WriteString("new file mode 100644\n")
		diff.WriteString("index 0000000..0000000\n")
		diff.WriteString("--- /dev/null\n")
		diff.WriteString(fmt.Sprintf("+++ b/%s\n", filePath))
	} else {
		// Modified file
		diff.WriteString("index 0000000..0000000 100644\n")
		diff.WriteString(fmt.Sprintf("--- a/%s\n", filePath))
		diff.WriteString(fmt.Sprintf("+++ b/%s\n", filePath))
	}

	oldLines := strings.Split(oldContent, "\n")
	newLines := strings.Split(newContent, "\n")

	// Simple line-by-line diff (this could be improved with a proper diff algorithm)
	maxLines := max(len(oldLines), len(newLines))

	if maxLines > 0 {
		startOld := 1
		startNew := 1
		if !oldFileExists {
			startOld = 0
		}
		diff.WriteString(fmt.Sprintf("@@ -%d,%d +%d,%d @@\n", startOld, len(oldLines), startNew, len(newLines)))

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

func (s *Service) GetGitConfig(repoPath string) (*models.GitConfig, error) {
	repo, err := s.OpenRepository(repoPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open repository: %w", err)
	}

	config, err := repo.Config()
	if err != nil {
		return nil, fmt.Errorf("failed to get repository config: %w", err)
	}

	return &models.GitConfig{
		Name:  config.User.Name,
		Email: config.User.Email,
	}, nil
}

func (s *Service) SetGitConfigIdentity(repoPath, name, email string) error {
	repo, err := s.OpenRepository(repoPath)
	if err != nil {
		return fmt.Errorf("failed to open repository: %w", err)
	}

	cfg, err := repo.Config()
	if err != nil {
		return fmt.Errorf("failed to get repository config: %w", err)
	}

	cfg.User.Name = name
	cfg.User.Email = email

	if err := repo.SetConfig(cfg); err != nil {
		return fmt.Errorf("failed to update repository config: %w", err)
	}

	return nil
}

func (s *Service) GetRemotes(repoPath string) ([]models.RepoRemote, error) {
	repo, err := s.OpenRepository(repoPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open repository: %w", err)
	}

	remotes, err := repo.Remotes()
	if err != nil {
		return nil, fmt.Errorf("failed to get remotes: %w", err)
	}

	result := make([]models.RepoRemote, 0, len(remotes))
	for _, remote := range remotes {
		cfg := remote.Config()
		url := ""
		if len(cfg.URLs) > 0 {
			url = cfg.URLs[0]
		}
		result = append(result, models.RepoRemote{
			Name: cfg.Name,
			URL:  url,
		})
	}

	return result, nil
}

// GetCommitFileDiff returns the diff for a specific file at a specific commit.
// It compares the file at the commit with its parent (or empty for initial commits).
func (s *Service) GetCommitFileDiff(repoPath, commitHash, filePath string, cursor, limit int) (*models.TokenizedDiff, error) {
	repo, err := s.OpenRepository(repoPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open repository: %w", err)
	}

	// Verify commit exists
	hash := plumbing.NewHash(commitHash)
	commit, err := repo.CommitObject(hash)
	if err != nil {
		return nil, fmt.Errorf("commit not found: %w", err)
	}

	// Get parent commit (if exists)
	var parentCommit *object.Commit
	if len(commit.ParentHashes) > 0 {
		parentCommit, _ = repo.CommitObject(commit.ParentHashes[0])
	}

	// Verify file exists in either commit or parent.
	commitTree, err := commit.Tree()
	if err != nil {
		return nil, fmt.Errorf("failed to get commit tree: %w", err)
	}

	var fileExists bool
	if _, err := commitTree.File(filePath); err == nil {
		fileExists = true
	}

	// Check file existence in parent (if exists).
	var oldFileExists bool
	if parentCommit != nil {
		parentTree, err := parentCommit.Tree()
		if err != nil {
			return nil, fmt.Errorf("failed to get parent tree: %w", err)
		}
		if _, err := parentTree.File(filePath); err == nil {
			oldFileExists = true
		}
	}

	// Handle file not found cases
	if !fileExists && !oldFileExists {
		return nil, fmt.Errorf("file not found in commit or parent: %s", filePath)
	}

	// Generate unified diff using git's native algorithm to avoid synthetic
	// delete/add pairs from naive line-by-line comparison.
	var cmd *exec.Cmd
	if parentCommit != nil {
		cmd = exec.Command(
			"git",
			"diff",
			parentCommit.Hash.String(),
			commit.Hash.String(),
			"--",
			filePath,
		)
	} else {
		// Initial commit has no parent, so show this commit's patch for the file.
		cmd = exec.Command(
			"git",
			"show",
			"--format=",
			"--patch",
			commit.Hash.String(),
			"--",
			filePath,
		)
	}
	cmd.Dir = repoPath

	output, err := cmd.Output()
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			stderr := strings.TrimSpace(string(exitErr.Stderr))
			if stderr != "" {
				return nil, fmt.Errorf("git diff failed: %s", stderr)
			}
		}
		return nil, fmt.Errorf("git diff failed: %w", err)
	}

	diffText := string(output)
	if strings.TrimSpace(diffText) == "" {
		return &models.TokenizedDiff{
			Filename:   filePath,
			Hunks:      []models.DiffHunkTokenized{},
			TotalHunks: 0,
			Additions:  0,
			Deletions:  0,
			HasMore:    false,
			NextCursor: 0,
		}, nil
	}

	// Tokenize and return
	return s.TokenizeDiff(diffText, filePath, cursor, limit), nil
}

// generateCommitFileDiff creates a unified diff between two file versions
func (s *Service) generateCommitFileDiff(filePath, oldContent string, oldExists bool, newContent string, newExists bool) string {
	// Handle file added (no parent content)
	if !oldExists && newExists {
		return s.generateNewFileDiff(filePath, newContent)
	}

	// Handle file deleted (no new content)
	if oldExists && !newExists {
		return s.generateDeletedFileDiff(filePath, oldContent)
	}

	// Handle modified file
	if oldContent == newContent {
		return ""
	}

	return s.generateTextDiff(filePath, oldContent, newContent)
}

// generateNewFileDiff creates diff for a new file
func (s *Service) generateNewFileDiff(filePath, content string) string {
	var diff strings.Builder
	diff.WriteString(fmt.Sprintf("diff --git a/%s b/%s\n", filePath, filePath))
	diff.WriteString("new file mode 100644\n")
	diff.WriteString("index 0000000..0000000\n")
	diff.WriteString("--- /dev/null\n")
	diff.WriteString(fmt.Sprintf("+++ b/%s\n", filePath))

	lines := strings.Split(content, "\n")
	if len(lines) > 0 && lines[len(lines)-1] == "" {
		lines = lines[:len(lines)-1]
	}

	if len(lines) > 0 {
		diff.WriteString(fmt.Sprintf("@@ -0,0 +1,%d @@\n", len(lines)))
		for _, line := range lines {
			diff.WriteString("+" + line + "\n")
		}
	}

	return diff.String()
}

// generateDeletedFileDiff creates diff for a deleted file
func (s *Service) generateDeletedFileDiff(filePath, content string) string {
	var diff strings.Builder
	diff.WriteString(fmt.Sprintf("diff --git a/%s b/%s\n", filePath, filePath))
	diff.WriteString("deleted file mode 100644\n")
	diff.WriteString("index 0000000..0000000\n")
	diff.WriteString(fmt.Sprintf("--- a/%s\n", filePath))
	diff.WriteString("+++ /dev/null\n")

	lines := strings.Split(content, "\n")
	if len(lines) > 0 && lines[len(lines)-1] == "" {
		lines = lines[:len(lines)-1]
	}

	if len(lines) > 0 {
		diff.WriteString(fmt.Sprintf("@@ -1,%d +0,0 @@\n", len(lines)))
		for _, line := range lines {
			diff.WriteString("-" + line + "\n")
		}
	}

	return diff.String()
}
