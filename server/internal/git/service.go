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