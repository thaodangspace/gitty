---
name: Commit Details Page Migration
description: Migrate commit detail view from modal dialog to dedicated page with URL routing
type: project
---

# Commit Details Page Migration - Design Spec

## Overview

Migrate the commit details view from a modal dialog overlay to a dedicated page with its own URL route. This improves shareability, bookmarkability, and browser navigation behavior.

## Current State

- Commit history is displayed at `/repo/:repoId`
- Clicking a commit opens `CommitDetailsDialog` as a modal overlay
- Modal state is not reflected in URL
- Page refresh loses modal state
- Browser back button closes modal (unnatural behavior)
- Limited screen space for viewing diffs

## Goals

1. **Direct URL access** - Each commit has a shareable, bookmarkable URL
2. **Natural navigation** - Browser back/forward buttons work as expected
3. **State persistence** - Refresh preserves the current commit view
4. **Feature parity** - Same content and functionality as modal
5. **Code reuse** - Shared component between dialog and page

## Non-Goals

- Adding new features beyond what the modal provides
- Changing the visual design of commit details content
- Modifying vim navigation behavior (Enter still activates)

## Architecture

### URL Structure

```
/repo/:repoId/commit/:commitHash
```

- Uses full 40-character commit hash for uniqueness
- Example: `/repo/my-repo/commit/a1b2c3d4e5f6g7h8i9j0k1l2m3n4o5p6q7r8s9t0`

### Component Refactoring

```
Before:
┌─────────────────────────────────────┐
│ CommitHistory.tsx                   │
│   └─ renders                        │
│       └─ CommitDetailsDialog.tsx    │
│           └─ Dialog content inline  │
└─────────────────────────────────────┘

After:
┌──────────────────────────────────────┐
│ CommitDetailsContent.tsx (NEW)       │
│   └─ Shared content component        │
│       ├─ Commit header               │
│       ├─ Stats summary               │
│       ├─ File change list            │
│       └─ InlineDiffViewer            │
└──────────────────────────────────────┘
         ▲
         │ used by
    ┌────┴────┐
    │         │
┌─────────┐  ┌──────────────────────┐
│ Dialog  │  │ CommitDetailsPage    │
│ wrapper │  │ (new route handler)  │
└─────────┘  └──────────────────────┘
```

### Routes

```
/                          → LandingPage
/repo/:repoId              → AppLayout (contains CommitHistory)
/repo/:repoId/commit/:hash → CommitDetailsPage (NEW)
```

## Component Details

### CommitDetailsContent.tsx (NEW)

Extracted from `CommitDetailsDialog.tsx`. Contains all commit detail rendering logic:

- Props: `commitHash: string | null`, `repositoryId: string`
- Uses existing `useCommitDetails` hook
- Renders:
  - Commit header (message, author, date, hash, parent hash)
  - Stats summary (additions, deletions, files changed)
  - File change list with expand/collapse
  - `InlineDiffViewer` for expanded files

### CommitDetailsDialog.tsx (MODIFIED)

- Wraps `CommitDetailsContent` in Dialog UI
- Dialog open state controlled by parent (`CommitHistory`)
- Minimal changes - mostly extraction of content

### CommitDetailsPage.tsx (NEW)

New page component for the commit details route:

- Renders `AppLayout` structure (header, main content, status bar)
- Includes back button navigating to `/repo/:repoId`
- Renders `CommitDetailsContent`
- Handles loading/error states

### CommitHistory.tsx (MODIFIED)

- Remove `CommitDetailsDialog` render
- Change click handlers from `setSelectedCommitHash(hash)` to `navigate(...)`
- Update vim navigation `onActivate` to navigate instead of opening modal

### App.tsx (MODIFIED)

- Add new route: `/repo/:repoId/commit/:commitHash` → `CommitDetailsPage`

## Navigation Flow

1. User on commit history page (`/repo/:repoId`)
2. Click commit card or press Enter (vim mode)
3. Navigate to `/repo/:repoId/commit/:hash`
4. `CommitDetailsPage` loads, fetches data
5. Click "Back to History" → navigate to `/repo/:repoId`

## Error Handling

- Missing/invalid commit hash → Show error state, offer back navigation
- Commit not found → Show error message, offer back navigation
- Network error → Standard error UI with retry option

## Testing Considerations

- Verify URL changes correctly on commit click
- Verify back button navigates to history
- Verify page refresh preserves commit view
- Verify vim navigation (Enter key) still works
- Verify dialog still works for any remaining use cases

## Implementation Order

1. Create `CommitDetailsContent.tsx` (extract from Dialog)
2. Update `CommitDetailsDialog.tsx` to use Content component
3. Create `CommitDetailsPage.tsx`
4. Add route to `App.tsx`
5. Update `CommitHistory.tsx` to navigate instead of open modal
6. Remove modal state from `CommitHistory`

## Success Criteria

- [ ] Clicking commit navigates to new page
- [ ] URL is shareable and bookmarkable
- [ ] Back button returns to commit history
- [ ] Dialog still functions (no regression)
- [ ] Vim navigation works (Enter opens page)
- [ ] All commit detail content renders correctly
- [ ] Inline diff viewer works in page context
