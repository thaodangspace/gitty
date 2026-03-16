package git

import (
	"testing"

	"gitweb/server/internal/models"
)

func TestFlushBlock_NilBlock(t *testing.T) {
	hunk := &models.DiffHunkTokenized{Blocks: []models.DiffBlock{}}
	result := flushBlock(nil, hunk)

	if result != nil {
		t.Error("Expected nil result for nil block")
	}
	if len(hunk.Blocks) != 0 {
		t.Error("Expected no blocks added for nil input")
	}
}

func TestFlushBlock_EmptyBlock(t *testing.T) {
	block := &models.DiffBlock{Type: "context", Lines: []models.DiffLineTokenized{}}
	hunk := &models.DiffHunkTokenized{Blocks: []models.DiffBlock{}}

	result := flushBlock(block, hunk)

	if result != nil {
		t.Error("Expected nil result")
	}
	if len(hunk.Blocks) != 0 {
		t.Error("Expected no blocks added for empty block")
	}
}

func TestFlushBlock_ContextBlockUnderThreshold(t *testing.T) {
	// 5 lines - should NOT collapse
	block := &models.DiffBlock{
		Type:  "context",
		Lines: make([]models.DiffLineTokenized, 5),
	}
	hunk := &models.DiffHunkTokenized{Blocks: []models.DiffBlock{}}

	flushBlock(block, hunk)

	if len(hunk.Blocks) != 1 {
		t.Fatal("Expected 1 block added")
	}
	if hunk.Blocks[0].Collapsed {
		t.Error("Context block with 5 lines should not be collapsed")
	}
}

func TestFlushBlock_ContextBlockAtThreshold(t *testing.T) {
	// 6 lines - should collapse
	block := &models.DiffBlock{
		Type:  "context",
		Lines: make([]models.DiffLineTokenized, 6),
	}
	hunk := &models.DiffHunkTokenized{Blocks: []models.DiffBlock{}}

	flushBlock(block, hunk)

	if len(hunk.Blocks) != 1 {
		t.Fatal("Expected 1 block added")
	}
	if !hunk.Blocks[0].Collapsed {
		t.Error("Context block with 6 lines should be collapsed")
	}
}

func TestFlushBlock_ContextBlockOverThreshold(t *testing.T) {
	// 10 lines - should collapse
	block := &models.DiffBlock{
		Type:  "context",
		Lines: make([]models.DiffLineTokenized, 10),
	}
	hunk := &models.DiffHunkTokenized{Blocks: []models.DiffBlock{}}

	flushBlock(block, hunk)

	if !hunk.Blocks[0].Collapsed {
		t.Error("Context block with 10 lines should be collapsed")
	}
}

func TestFlushBlock_AddedBlockNeverCollapses(t *testing.T) {
	block := &models.DiffBlock{
		Type:  "added",
		Lines: make([]models.DiffLineTokenized, 20),
	}
	hunk := &models.DiffHunkTokenized{Blocks: []models.DiffBlock{}}

	flushBlock(block, hunk)

	if hunk.Blocks[0].Collapsed {
		t.Error("Added block should never be collapsed")
	}
}

func TestFlushBlock_DeletedBlockNeverCollapses(t *testing.T) {
	block := &models.DiffBlock{
		Type:  "deleted",
		Lines: make([]models.DiffLineTokenized, 20),
	}
	hunk := &models.DiffHunkTokenized{Blocks: []models.DiffBlock{}}

	flushBlock(block, hunk)

	if hunk.Blocks[0].Collapsed {
		t.Error("Deleted block should never be collapsed")
	}
}

func TestTokenizeDiff_SingleLinesSeparateBlocks(t *testing.T) {
	service := NewService()

	// Diff with alternating line types
	diff := `diff --git a/test.js b/test.js
index 1234567..abcdefg 100644
--- a/test.js
+++ b/test.js
@@ -1,3 +1,3 @@
 const x = 1;
-const y = 2;
+const y = 3;
 const z = 4;`

	result := service.TokenizeDiff(diff, "test.js", 0, 50)

	if len(result.Hunks) == 0 {
		t.Fatal("Expected at least one hunk")
	}

	hunk := result.Hunks[0]
	// Context, deleted, added, context = 4 blocks
	if len(hunk.Blocks) != 4 {
		t.Errorf("Expected 4 blocks (context, deleted, added, context), got %d", len(hunk.Blocks))
	}
}

func TestTokenizeDiff_ConsecutiveSameTypeSingleBlock(t *testing.T) {
	service := NewService()

	// Diff with consecutive added lines
	diff := `diff --git a/test.js b/test.js
index 1234567..abcdefg 100644
--- a/test.js
+++ b/test.js
@@ -1,1 +1,3 @@
 const x = 1;
+const y = 2;
+const z = 3;`

	result := service.TokenizeDiff(diff, "test.js", 0, 50)

	if len(result.Hunks) == 0 {
		t.Fatal("Expected at least one hunk")
	}

	hunk := result.Hunks[0]
	// Context, added(2) = 2 blocks
	if len(hunk.Blocks) != 2 {
		t.Errorf("Expected 2 blocks (context, added), got %d", len(hunk.Blocks))
	}

	// Second block should have 2 lines
	if len(hunk.Blocks[1].Lines) != 2 {
		t.Errorf("Expected added block to have 2 lines, got %d", len(hunk.Blocks[1].Lines))
	}
	if hunk.Blocks[1].Type != "added" {
		t.Errorf("Expected second block type 'added', got %s", hunk.Blocks[1].Type)
	}
}

func TestTokenizeDiff_ContextBlockCollapsedAtThreshold(t *testing.T) {
	service := NewService()

	// Diff with 6 context lines
	diff := `diff --git a/test.js b/test.js
index 1234567..abcdefg 100644
--- a/test.js
+++ b/test.js
@@ -1,6 +1,6 @@
 line1
 line2
 line3
 line4
 line5
 line6`

	result := service.TokenizeDiff(diff, "test.js", 0, 50)

	if len(result.Hunks) == 0 {
		t.Fatal("Expected at least one hunk")
	}

	hunk := result.Hunks[0]
	if len(hunk.Blocks) != 1 {
		t.Fatalf("Expected 1 block, got %d", len(hunk.Blocks))
	}

	if !hunk.Blocks[0].Collapsed {
		t.Error("Context block with 6 lines should be collapsed")
	}
}

func TestTokenizeDiff_LineNumberRanges(t *testing.T) {
	service := NewService()

	// Diff with multiple added lines to test line number ranges
	// Using hunk header starting at line 10
	diff := `diff --git a/test.js b/test.js
index 1234567..abcdefg 100644
--- a/test.js
+++ b/test.js
@@ -10,2 +10,4 @@
 const x = 1;
+const y = 2;
+const z = 3;`

	result := service.TokenizeDiff(diff, "test.js", 0, 50)

	hunk := result.Hunks[0]
	addedBlock := hunk.Blocks[1] // Second block should be added

	if addedBlock.Type != "added" {
		t.Fatalf("Expected 'added' type, got %s", addedBlock.Type)
	}

	// The added lines are at positions 11 and 12 in the new file
	// (context line is at line 10, added lines follow)
	// Note: Line numbers are tracked from hunk header; context is line 10,
	// so added lines start at 11 and end at 12
	if addedBlock.StartNew != 11 {
		t.Errorf("Expected StartNew=11, got %d", addedBlock.StartNew)
	}

	// EndNew should be 12 (last added line)
	if addedBlock.EndNew != 12 {
		t.Errorf("Expected EndNew=12, got %d", addedBlock.EndNew)
	}

	// StartOld/EndOld should be 0 for added block
	if addedBlock.StartOld != 0 || addedBlock.EndOld != 0 {
		t.Errorf("Added block should have StartOld=0 and EndOld=0, got %d/%d", addedBlock.StartOld, addedBlock.EndOld)
	}
}

func TestTokenizeDiff_MultipleHunksBlockBoundary(t *testing.T) {
	service := NewService()

	// Diff with multiple hunks to verify block flush at hunk boundaries
	diff := `diff --git a/test.js b/test.js
index 1234567..abcdefg 100644
--- a/test.js
+++ b/test.js
@@ -1,2 +1,3 @@
 const x = 1;
+const y = 2;
 const z = 3;
@@ -10,2 +11,3 @@
 const a = 1;
+const b = 2;
 const c = 3;`

	result := service.TokenizeDiff(diff, "test.js", 0, 50)

	if len(result.Hunks) != 2 {
		t.Fatalf("Expected 2 hunks, got %d", len(result.Hunks))
	}

	// First hunk: context, added, context = 3 blocks
	if len(result.Hunks[0].Blocks) != 3 {
		t.Errorf("Expected first hunk to have 3 blocks, got %d", len(result.Hunks[0].Blocks))
	}

	// Second hunk: context, added, context = 3 blocks
	if len(result.Hunks[1].Blocks) != 3 {
		t.Errorf("Expected second hunk to have 3 blocks, got %d", len(result.Hunks[1].Blocks))
	}
}

func TestTokenizeDiff_RealisticDiff(t *testing.T) {
	service := NewService()

	// Realistic diff with multiple block types and a collapsed context block
	diff := `diff --git a/app.ts b/app.ts
index 1234567..abcdefg 100644
--- a/app.ts
+++ b/app.ts
@@ -5,15 +5,12 @@
 import React from 'react';
 import { View } from 'react-native';
 import { Button } from './Button';
-import { OldComponent } from './OldComponent';
-import { LegacyUtil } from './LegacyUtil';
+import { NewComponent } from './NewComponent';
 import { Helper } from './Helper';
 import { Config } from './Config';
 import { Logger } from './Logger';
 import { Analytics } from './Analytics';
-import { Deprecated } from './Deprecated';
-import { Removed } from './Removed';
+import { Fresh } from './Fresh';
 import { Final } from './Final';

 export function App() {`

	result := service.TokenizeDiff(diff, "app.ts", 0, 50)

	// Verify overall structure
	if len(result.Hunks) != 1 {
		t.Fatalf("Expected 1 hunk, got %d", len(result.Hunks))
	}

	hunk := result.Hunks[0]

	// Should have at least 3 blocks
	if len(hunk.Blocks) < 3 {
		t.Errorf("Expected at least 3 blocks, got %d", len(hunk.Blocks))
	}

	// Verify additions/deletions counts
	if result.Additions != 2 {
		t.Errorf("Expected 2 additions, got %d", result.Additions)
	}
	if result.Deletions != 4 {
		t.Errorf("Expected 4 deletions, got %d", result.Deletions)
	}
}