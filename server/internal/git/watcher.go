package git

import (
	"log"
	"path/filepath"
	"sync"
	"time"

	"github.com/fsnotify/fsnotify"
)

// RepositoryWatcher manages file system watchers for git repositories
type RepositoryWatcher struct {
	watcher     *fsnotify.Watcher
	subscribers map[string][]chan struct{}
	mu          sync.RWMutex
	watchedDirs map[string]bool
}

// NewRepositoryWatcher creates a new repository watcher
func NewRepositoryWatcher() (*RepositoryWatcher, error) {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, err
	}

	rw := &RepositoryWatcher{
		watcher:     watcher,
		subscribers: make(map[string][]chan struct{}),
		watchedDirs: make(map[string]bool),
	}

	go rw.watchLoop()

	return rw, nil
}

// watchLoop processes file system events
func (rw *RepositoryWatcher) watchLoop() {
	for {
		select {
		case event, ok := <-rw.watcher.Events:
			if !ok {
				return
			}

			// Notify all subscribers for this repository path
			if event.Op&(fsnotify.Write|fsnotify.Create|fsnotify.Remove|fsnotify.Rename) != 0 {
				rw.notifySubscribers(event.Name)
			}

		case err, ok := <-rw.watcher.Errors:
			if !ok {
				return
			}
			log.Printf("Watcher error: %v", err)
		}
	}
}

// notifySubscribers notifies all subscribers for a given repository
func (rw *RepositoryWatcher) notifySubscribers(path string) {
	rw.mu.RLock()
	defer rw.mu.RUnlock()

	// Find the repository path this file belongs to
	for repoPath, subscribers := range rw.subscribers {
		// Check if the changed file is under this repository
		if isUnderPath(path, repoPath) {
			// Notify all subscribers (non-blocking)
			for _, ch := range subscribers {
				select {
				case ch <- struct{}{}:
				default:
					// Channel full or closed, skip
				}
			}
		}
	}
}

// Subscribe creates a subscription for repository changes
func (rw *RepositoryWatcher) Subscribe(repoPath string) (<-chan struct{}, func(), error) {
	rw.mu.Lock()
	defer rw.mu.Unlock()

	// Ensure the repository directory is being watched
	if !rw.watchedDirs[repoPath] {
		// Watch the repository root
		if err := rw.watcher.Add(repoPath); err != nil {
			return nil, nil, err
		}

		// Watch the .git directory for ref changes, index updates, etc.
		gitDir := filepath.Join(repoPath, ".git")
		if err := rw.watcher.Add(gitDir); err != nil {
			// If .git is not a directory (could be a file in submodules), ignore the error
			log.Printf("Warning: Could not watch .git directory: %v", err)
		}

		rw.watchedDirs[repoPath] = true
	}

	// Create a buffered channel to avoid blocking the notifier
	ch := make(chan struct{}, 1)

	// Add subscriber
	rw.subscribers[repoPath] = append(rw.subscribers[repoPath], ch)

	// Return unsubscribe function
	unsubscribe := func() {
		rw.mu.Lock()
		defer rw.mu.Unlock()

		subscribers := rw.subscribers[repoPath]
		for i, sub := range subscribers {
			if sub == ch {
				// Remove this subscriber
				rw.subscribers[repoPath] = append(subscribers[:i], subscribers[i+1:]...)
				close(ch)
				break
			}
		}

		// If no more subscribers, stop watching this directory
		if len(rw.subscribers[repoPath]) == 0 {
			delete(rw.subscribers, repoPath)
			rw.watcher.Remove(repoPath)
			rw.watcher.Remove(filepath.Join(repoPath, ".git"))
			delete(rw.watchedDirs, repoPath)
		}
	}

	return ch, unsubscribe, nil
}

// WaitForChange waits for a change notification or timeout
func (rw *RepositoryWatcher) WaitForChange(repoPath string, timeout time.Duration) bool {
	ch, unsubscribe, err := rw.Subscribe(repoPath)
	if err != nil {
		log.Printf("Failed to subscribe to repository changes: %v", err)
		return false
	}
	defer unsubscribe()

	timer := time.NewTimer(timeout)
	defer timer.Stop()

	select {
	case <-ch:
		return true // Change detected
	case <-timer.C:
		return false // Timeout
	}
}

// Close shuts down the watcher
func (rw *RepositoryWatcher) Close() error {
	rw.mu.Lock()
	defer rw.mu.Unlock()

	// Close all subscriber channels
	for _, subscribers := range rw.subscribers {
		for _, ch := range subscribers {
			close(ch)
		}
	}

	rw.subscribers = make(map[string][]chan struct{})
	rw.watchedDirs = make(map[string]bool)

	return rw.watcher.Close()
}

// isUnderPath checks if a file path is under a given directory path
func isUnderPath(filePath, dirPath string) bool {
	rel, err := filepath.Rel(dirPath, filePath)
	if err != nil {
		return false
	}
	// If the relative path starts with "..", the file is outside the directory
	return len(rel) > 0 && rel[0] != '.' && rel[:2] != ".."
}
