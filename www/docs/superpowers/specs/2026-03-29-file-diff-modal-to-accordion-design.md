# File Diff Viewer: Modal to Accordion Migration

**Date:** 2026-03-29
**Author:** Claude
**Status:** Approved

## Overview

Migrate the file diff viewer from a modal overlay pattern to an inline accordion/collapse-expand pattern within the Working Directory Changes view.

## Problem Statement

The current implementation opens a modal overlay when clicking "View changes" on a file. This breaks context, requires closing the modal to view another file, and creates overlay fatigue for users who frequently switch between file diffs.

## Goals

- Keep users in context while viewing diffs
- Enable quick switching between file diffs
- Maintain all existing diff functionality (syntax highlighting, load more hunks, etc.)
- Support single expanded diff at a time (accordion behavior)

## Architecture

### Component Hierarchy

```
WorkingDirectoryChanges.tsx
├── FileChangeItem (updated with onClick toggle)
│   └── Expanded state → renders InlineDiffViewer below
├── InlineDiffViewer (new component)
│   └── Uses shared DiffContent logic without modal wrapper
└── State: expandedFilePath (string | null)
```

### State Management

**New atom in `ui-atoms.ts`:**
```typescript
export const expandedFilePathAtom = atom<string | null>(null);
```

This tracks which file path is currently expanded. `null` means no file is expanded.

### Component Refactoring

**1. Extract `DiffContent` from `DiffViewer.tsx`**

The current `DiffViewer` component mixes modal container logic with diff fetching/rendering logic. We extract the rendering logic:

```tsx
// DiffContent.tsx (new shared component)
interface DiffContentProps {
    repositoryId: string;
    filePath: string;
    commitHash?: string;
}

// Handles:
// - Fetching diff (useQuery or inline useEffect)
// - Loading states
// - Error states
// - Rendering TokenizedDiffRenderer
// - Load more hunks functionality
```

**2. `DiffViewer.tsx` (modal version)**

```tsx
// Wraps DiffContent with modal overlay
// Used by: CommitDetailsDialog, other modal use cases
```

**3. `InlineDiffViewer.tsx` (new inline version)**

```tsx
// Renders DiffContent without modal wrapper
// Styled for inline display (no max-h constraint, full width)
// Used by: WorkingDirectoryChanges
```

## Implementation Details

### File: `src/store/atoms/ui-atoms.ts`

Add the expanded file path atom:

```typescript
export const expandedFilePathAtom = atom<string | null>(null);
```

### File: `src/components/file/DiffContent.tsx` (new)

Extract the diff logic from `DiffViewer`:
- `useIntersectionObserver` for load-more functionality
- `loadMore` callback for pagination
- Loading/error/empty states
- `TokenizedDiffRenderer` integration

### File: `src/components/file/DiffViewer.tsx`

Refactor to use `DiffContent`:

```tsx
export default function DiffViewer({ onClose, ...diffProps }: DiffViewerProps) {
    return (
        <div className="fixed inset-0 z-50 bg-black/50 flex items-center justify-center p-4">
            <div className="bg-white rounded-lg shadow-xl max-w-6xl w-full max-h-[90vh] flex flex-col">
                {/* Header with close button */}
                {/* DiffContent {...diffProps} /> */}
            </div>
        </div>
    );
}
```

### File: `src/components/file/InlineDiffViewer.tsx` (new)

```tsx
interface InlineDiffViewerProps {
    repositoryId: string;
    filePath: string;
    fileName: string;
    commitHash?: string;
}

export default function InlineDiffViewer({ repositoryId, filePath, fileName, commitHash }: InlineDiffViewerProps) {
    return (
        <div className="mt-3 border rounded-lg bg-muted/30">
            <div className="flex items-center justify-between p-3 border-b">
                <div>
                    <h3 className="text-sm font-medium">{fileName}</h3>
                    <p className="text-xs text-muted-foreground">{filePath}</p>
                </div>
            </div>
            <div className="p-3">
                <DiffContent repositoryId={repositoryId} filePath={filePath} commitHash={commitHash} />
            </div>
        </div>
    );
}
```

### File: `src/components/repository/WorkingDirectoryChanges.tsx`

**State change:**
```tsx
// Replace:
const [selectedDiffFile, setSelectedDiffFile] = useState<{ path: string; name: string } | null>(null);

// With:
const [expandedFilePath, setExpandedFilePath] = useAtom(expandedFilePathAtom);
```

**FileChangeItem updates:**
```tsx
<FileChangeItem
    file={file}
    isStaged={false}
    onStage={() => handleStageFile(file.path)}
    onExpand={() => {
        setExpandedFilePath(expandedFilePath === file.path ? null : file.path);
    }}
    isExpanded={expandedFilePath === file.path}
    isVimFocused={isVimActive && currentIndex === globalIndex}
/>
```

**Render inline diff below each file row:**
```tsx
{expandedFilePath === file.path && (
    <InlineDiffViewer
        repositoryId={currentRepository.id}
        filePath={file.path}
        fileName={file.path.split('/').pop() || file.path}
    />
)}
```

## Visual Design

### Expanded State Indicators

- **File row:** Subtle background highlight (`bg-muted/50`) when expanded
- **Chevron icon:** Rotate 90° when expanded (optional, if added to file row)
- **Border emphasis:** Slightly darker border on expanded file row

### Inline Diff Container

- Full width within the list container
- No height constraint (scrolls with the page)
- Muted background to distinguish from file rows
- Compact header (filename + path, no action buttons needed)

## Interaction Patterns

| Action | Result |
|--------|--------|
| Click file row (non-button area) | Toggle expand/collapse |
| Click action button (Stage/View) | Execute action, no expand toggle |
| Expand file A while B is expanded | Collapse B, expand A |
| Expand file A while A is expanded | Collapse A |
| Click outside file list | No change (diff stays inline) |

## Error Handling

Preserve all existing error handling from `DiffViewer`:
- Loading state with spinner
- Error state with message
- Empty diff state ("No changes to display")
- Load more hunks pagination

## Testing Considerations

1. **Functional tests:**
   - Expand/collapse toggles correctly
   - Only one diff expanded at a time
   - Load more hunks works inline
   - Stage/Unstage actions work without collapsing

2. **Visual tests:**
   - Expanded state is visually distinct
   - No layout shift when expanding/collapsing
   - Diff content is readable at full width

3. **Performance tests:**
   - No unnecessary re-renders when expanding/collapsing
   - Diff fetching only occurs when expanded (lazy loading)

## Migration Path

1. Add `expandedFilePathAtom` to state
2. Create `DiffContent` shared component
3. Refactor `DiffViewer` to use `DiffContent`
4. Create `InlineDiffViewer` component
5. Update `WorkingDirectoryChanges` to use new pattern
6. Remove modal rendering from `WorkingDirectoryChanges`

## Future Considerations (Out of Scope)

- Multiple simultaneous expanded diffs (configurable limit)
- Animated expand/collapse transitions
- Keyboard shortcuts for expand/collapse navigation
- Persisting expanded state across page refreshes
