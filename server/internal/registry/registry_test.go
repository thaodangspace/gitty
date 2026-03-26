package registry

import (
	"path/filepath"
	"testing"
	"time"
)

func TestNewCreatesFileIfMissing(t *testing.T) {
	tmp := t.TempDir()
	path := filepath.Join(tmp, "config", "gitty", "repository.json")

	reg, err := New(path)
	if err != nil {
		t.Fatalf("New() returned error: %v", err)
	}

	entries := reg.List()
	if len(entries) != 0 {
		t.Fatalf("expected empty list, got %d entries", len(entries))
	}
}

func TestAddAndList(t *testing.T) {
	tmp := t.TempDir()
	path := filepath.Join(tmp, "repository.json")

	reg, err := New(path)
	if err != nil {
		t.Fatalf("New() error: %v", err)
	}

	entry := Entry{
		ID:         "my-repo",
		Name:       "my-repo",
		Path:       "/home/user/projects/my-repo",
		Source:     "imported",
		ImportedAt: time.Now(),
	}

	if err := reg.Add(entry); err != nil {
		t.Fatalf("Add() error: %v", err)
	}

	entries := reg.List()
	if len(entries) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(entries))
	}
	if entries[0].ID != "my-repo" {
		t.Fatalf("expected ID 'my-repo', got %q", entries[0].ID)
	}
}

func TestAddDuplicateIDErrors(t *testing.T) {
	tmp := t.TempDir()
	path := filepath.Join(tmp, "repository.json")

	reg, _ := New(path)

	entry := Entry{ID: "dup", Name: "dup", Path: "/a", Source: "imported", ImportedAt: time.Now()}
	if err := reg.Add(entry); err != nil {
		t.Fatalf("first Add() error: %v", err)
	}

	entry2 := Entry{ID: "dup", Name: "dup2", Path: "/b", Source: "imported", ImportedAt: time.Now()}
	if err := reg.Add(entry2); err == nil {
		t.Fatal("expected error for duplicate ID")
	}
}

func TestAddDuplicatePathErrors(t *testing.T) {
	tmp := t.TempDir()
	path := filepath.Join(tmp, "repository.json")

	reg, _ := New(path)

	entry := Entry{ID: "a", Name: "a", Path: "/same/path", Source: "imported", ImportedAt: time.Now()}
	reg.Add(entry)

	entry2 := Entry{ID: "b", Name: "b", Path: "/same/path", Source: "imported", ImportedAt: time.Now()}
	if err := reg.Add(entry2); err == nil {
		t.Fatal("expected error for duplicate path")
	}
}

func TestGet(t *testing.T) {
	tmp := t.TempDir()
	path := filepath.Join(tmp, "repository.json")

	reg, _ := New(path)
	reg.Add(Entry{ID: "foo", Name: "foo", Path: "/foo", Source: "created", ImportedAt: time.Now()})

	entry, ok := reg.Get("foo")
	if !ok {
		t.Fatal("expected to find entry")
	}
	if entry.Path != "/foo" {
		t.Fatalf("expected path '/foo', got %q", entry.Path)
	}

	_, ok = reg.Get("nonexistent")
	if ok {
		t.Fatal("expected not to find entry")
	}
}

func TestRemove(t *testing.T) {
	tmp := t.TempDir()
	path := filepath.Join(tmp, "repository.json")

	reg, _ := New(path)
	reg.Add(Entry{ID: "rm-me", Name: "rm-me", Path: "/rm", Source: "imported", ImportedAt: time.Now()})

	if err := reg.Remove("rm-me"); err != nil {
		t.Fatalf("Remove() error: %v", err)
	}

	if len(reg.List()) != 0 {
		t.Fatal("expected empty list after remove")
	}
}

func TestRemoveNonexistentErrors(t *testing.T) {
	tmp := t.TempDir()
	path := filepath.Join(tmp, "repository.json")

	reg, _ := New(path)

	if err := reg.Remove("nope"); err == nil {
		t.Fatal("expected error for nonexistent entry")
	}
}

func TestPersistenceSurvivesReload(t *testing.T) {
	tmp := t.TempDir()
	path := filepath.Join(tmp, "repository.json")

	reg, _ := New(path)
	reg.Add(Entry{ID: "persist", Name: "persist", Path: "/persist", Source: "cloned", ImportedAt: time.Now()})

	// Reload from disk
	reg2, err := New(path)
	if err != nil {
		t.Fatalf("reload error: %v", err)
	}

	entries := reg2.List()
	if len(entries) != 1 || entries[0].ID != "persist" {
		t.Fatalf("expected persisted entry, got %v", entries)
	}
}
