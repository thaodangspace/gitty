package registry

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// Entry represents a repository reference persisted to disk.
type Entry struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	Path        string    `json:"path"`
	URL         string    `json:"url,omitempty"`
	Description string    `json:"description,omitempty"`
	Source      string    `json:"source"`      // "imported" | "cloned" | "created"
	ImportedAt  time.Time `json:"imported_at"`
}

// Registry manages persistent repository references stored in a JSON file.
type Registry struct {
	mu      sync.RWMutex
	path    string
	entries []Entry
}

// New loads or creates the registry file at path. Parent directories are created if missing.
func New(path string) (*Registry, error) {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, fmt.Errorf("create registry directory: %w", err)
	}

	r := &Registry{path: path}

	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			r.entries = []Entry{}
			return r, nil
		}
		return nil, fmt.Errorf("read registry file: %w", err)
	}

	if err := json.Unmarshal(data, &r.entries); err != nil {
		return nil, fmt.Errorf("decode registry file: %w", err)
	}

	return r, nil
}

// List returns all registry entries.
func (r *Registry) List() []Entry {
	r.mu.RLock()
	defer r.mu.RUnlock()

	out := make([]Entry, len(r.entries))
	copy(out, r.entries)
	return out
}

// Get returns the entry with the given ID.
func (r *Registry) Get(id string) (*Entry, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	for i := range r.entries {
		if r.entries[i].ID == id {
			e := r.entries[i]
			return &e, true
		}
	}
	return nil, false
}

// Add persists a new entry. Returns an error if the ID or path already exists.
func (r *Registry) Add(entry Entry) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	for _, e := range r.entries {
		if e.ID == entry.ID {
			return fmt.Errorf("registry: duplicate ID %q", entry.ID)
		}
		if e.Path == entry.Path {
			return fmt.Errorf("registry: duplicate path %q", entry.Path)
		}
	}

	r.entries = append(r.entries, entry)
	return r.save()
}

// Remove deletes the entry with the given ID. Returns an error if not found.
func (r *Registry) Remove(id string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	for i, e := range r.entries {
		if e.ID == id {
			r.entries = append(r.entries[:i], r.entries[i+1:]...)
			return r.save()
		}
	}
	return fmt.Errorf("registry: entry %q not found", id)
}

// save writes the entries to disk atomically (temp file + rename).
func (r *Registry) save() error {
	data, err := json.MarshalIndent(r.entries, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal registry: %w", err)
	}

	tmp := r.path + ".tmp"
	if err := os.WriteFile(tmp, data, 0o644); err != nil {
		return fmt.Errorf("write temp registry file: %w", err)
	}

	if err := os.Rename(tmp, r.path); err != nil {
		os.Remove(tmp)
		return fmt.Errorf("rename registry file: %w", err)
	}

	return nil
}
