# Commit Details Page Migration Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Migrate commit detail view from modal dialog to a dedicated page with URL routing at `/repo/:repoId/commit/:commitHash`

**Architecture:** Extract shared `CommitDetailsContent` component from `CommitDetailsDialog`, create new `CommitDetailsPage` component, add route, update `CommitHistory` to navigate instead of opening modal

**Tech Stack:** React, TypeScript, react-router-dom, Tailwind CSS, Radix UI

---

## File Structure

| File | Action | Responsibility |
|------|--------|----------------|
| `src/components/repository/CommitDetailsContent.tsx` | Create | Shared content component (extracted from Dialog) |
| `src/components/repository/CommitDetailsDialog.tsx` | Modify | Wrap Content in Dialog UI |
| `src/pages/CommitDetailsPage.tsx` | Create | Page component for commit details route |
| `src/App.tsx` | Modify | Add new route |
| `src/components/repository/CommitHistory.tsx` | Modify | Navigate to commit page instead of opening modal |

---

### Task 1: Create CommitDetailsContent component (shared)

**Files:**
- Create: `src/components/repository/CommitDetailsContent.tsx`

- [ ] **Step 1: Create CommitDetailsContent component**

```tsx
import { useAtom } from "jotai";
import { useState } from "react";
import { selectedRepositoryAtom } from "@/store/atoms";
import { useCommitDetails } from "@/store/queries";
import { format } from "date-fns";
import {
  GitCommit,
  User,
  Calendar,
  Hash,
  Plus,
  Minus,
  FileText,
  ChevronDown,
  ChevronUp,
} from "lucide-react";
import { Badge } from "@/components/ui/badge";
import InlineDiffViewer from "@/components/file/InlineDiffViewer";

export interface CommitDetailsContentProps {
  commitHash: string | null;
}

export default function CommitDetailsContent({ commitHash }: CommitDetailsContentProps) {
  const [currentRepository] = useAtom(selectedRepositoryAtom);
  const {
    data: commitDetails,
    isLoading,
    error,
  } = useCommitDetails(currentRepository?.id, commitHash || undefined);
  const [expandedFilePath, setExpandedFilePath] = useState<string | null>(null);

  const getChangeTypeColor = (changeType: string) => {
    switch (changeType) {
      case "added":
        return "bg-green-100 text-green-800 border-green-200";
      case "deleted":
        return "bg-red-100 text-red-800 border-red-200";
      case "modified":
        return "bg-blue-100 text-blue-800 border-blue-200";
      default:
        return "bg-gray-100 text-gray-800 border-gray-200";
    }
  };

  const getChangeTypeIcon = (changeType: string) => {
    switch (changeType) {
      case "added":
        return <Plus className="h-3 w-3" />;
      case "deleted":
        return <Minus className="h-3 w-3" />;
      case "modified":
        return <FileText className="h-3 w-3" />;
      default:
        return <FileText className="h-3 w-3" />;
    }
  };

  const handleFileClick = (filePath: string) => {
    setExpandedFilePath((current) =>
      current === filePath ? null : filePath
    );
  };

  if (isLoading) {
    return (
      <div className="p-8 flex items-center justify-center">
        <div className="text-muted-foreground">Loading commit details...</div>
      </div>
    );
  }

  if (error) {
    return (
      <div className="p-8 flex items-center justify-center">
        <div className="text-red-600">
          Error loading commit details: {error.message}
        </div>
      </div>
    );
  }

  if (!commitDetails) {
    return (
      <div className="p-8 flex items-center justify-center">
        <div className="text-muted-foreground">Commit not found</div>
      </div>
    );
  }

  return (
    <div className="flex-1 overflow-auto">
      {/* Commit Header */}
      <div className="p-6 border-b bg-muted/30">
        <div className="flex items-start gap-4">
          <div className="flex-shrink-0 mt-1">
            <div className="w-10 h-10 bg-primary/10 rounded-full flex items-center justify-center">
              <GitCommit className="h-5 w-5 text-primary" />
            </div>
          </div>

          <div className="flex-1 min-w-0">
            <h3 className="text-lg font-semibold mb-3 leading-relaxed">
              {commitDetails.message}
            </h3>

            <div className="flex items-center gap-6 text-sm text-muted-foreground mb-4">
              <div className="flex items-center gap-2">
                <User className="h-4 w-4" />
                <span className="font-medium">
                  {commitDetails.author.name}
                </span>
                <span className="text-muted-foreground">
                  ({commitDetails.author.email})
                </span>
              </div>
              <div className="flex items-center gap-2">
                <Calendar className="h-4 w-4" />
                <span>
                  {format(
                    new Date(commitDetails.date),
                    "MMM d, yyyy HH:mm:ss",
                  )}
                </span>
              </div>
            </div>

            <div className="flex items-center gap-4">
              <div className="flex items-center gap-2 text-sm bg-muted px-3 py-1 rounded font-mono">
                <Hash className="h-3 w-3" />
                <span>{commitDetails.hash.substring(0, 12)}</span>
              </div>
              {commitDetails.parent_hash && (
                <div className="text-sm text-muted-foreground">
                  Parent:{" "}
                  <span className="font-mono">
                    {commitDetails.parent_hash.substring(0, 12)}
                  </span>
                </div>
              )}
            </div>
          </div>
        </div>
      </div>

      {/* Stats Summary */}
      <div className="p-6 border-b bg-background">
        <div className="flex items-center gap-6">
          <div className="flex items-center gap-2">
            <Badge
              variant="outline"
              className="bg-green-50 text-green-700 border-green-200"
            >
              <Plus className="h-3 w-3 mr-1" />+
              {commitDetails.stats.additions}
            </Badge>
            <Badge
              variant="outline"
              className="bg-red-50 text-red-700 border-red-200"
            >
              <Minus className="h-3 w-3 mr-1" />-
              {commitDetails.stats.deletions}
            </Badge>
          </div>
          <div className="text-sm text-muted-foreground">
            {commitDetails.stats.files_changed} file
            {commitDetails.stats.files_changed !== 1 ? "s" : ""} changed
          </div>
        </div>
      </div>

      {/* File Changes */}
      <div className="p-6">
        <h4 className="text-md font-semibold mb-4">
          Files Changed ({commitDetails.changes.length})
        </h4>

        <div className="space-y-2">
          {commitDetails.changes.map((change) => {
            const isExpanded = expandedFilePath === change.path;
            return (
              <div
                key={change.path}
                className={`border rounded-lg overflow-hidden ${
                  isExpanded ? "ring-1 ring-primary/20" : ""
                }`}
              >
                <button
                  type="button"
                  aria-expanded={isExpanded}
                  onClick={() => handleFileClick(change.path)}
                  className={`w-full flex items-center justify-between p-4 text-left transition-colors ${
                    isExpanded
                      ? "bg-muted/50 border-b"
                      : "bg-muted/30 hover:bg-muted/50"
                  }`}
                >
                  <div className="flex items-center gap-3 min-w-0">
                    <Badge
                      variant="outline"
                      className={`${getChangeTypeColor(change.change_type)} font-mono text-xs flex-shrink-0`}
                    >
                      {getChangeTypeIcon(change.change_type)}
                      {change.change_type}
                    </Badge>
                    <span className="font-mono text-sm truncate">
                      {change.path}
                    </span>
                  </div>
                  <div className="flex items-center gap-3 flex-shrink-0 ml-2">
                    {(change.additions > 0 || change.deletions > 0) && (
                      <div className="flex items-center gap-2 text-xs">
                        {change.additions > 0 && (
                          <span className="text-green-600">
                            +{change.additions}
                          </span>
                        )}
                        {change.deletions > 0 && (
                          <span className="text-red-600">
                            -{change.deletions}
                          </span>
                        )}
                      </div>
                    )}
                    {isExpanded ? (
                      <ChevronUp className="h-4 w-4 text-muted-foreground" />
                    ) : (
                      <ChevronDown className="h-4 w-4 text-muted-foreground" />
                    )}
                  </div>
                </button>

                {isExpanded && currentRepository && (
                  <div className="border-t">
                    <InlineDiffViewer
                      repositoryId={currentRepository.id}
                      filePath={change.path}
                      commitHash={commitDetails.hash}
                    />
                  </div>
                )}
              </div>
            );
          })}
        </div>
      </div>
    </div>
  );
}
```

- [ ] **Step 2: Commit**

```bash
git add src/components/repository/CommitDetailsContent.tsx
git commit -m "feat: extract CommitDetailsContent shared component

Extract commit detail rendering logic into reusable component
for use by both dialog and page components.

Co-Authored-By: Claude Opus 4.6 <noreply@anthropic.com>
"
```

---

### Task 2: Update CommitDetailsDialog to use CommitDetailsContent

**Files:**
- Modify: `src/components/repository/CommitDetailsDialog.tsx`

- [ ] **Step 1: Update CommitDetailsDialog to wrap CommitDetailsContent**

Replace the entire file content with:

```tsx
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import { GitCommit } from "lucide-react";
import CommitDetailsContent, { CommitDetailsContentProps } from "./CommitDetailsContent";

interface CommitDetailsDialogProps extends CommitDetailsContentProps {
  isOpen: boolean;
  onClose: () => void;
}

export default function CommitDetailsDialog({
  commitHash,
  isOpen,
  onClose,
}: CommitDetailsDialogProps) {
  return (
    <Dialog open={isOpen} onOpenChange={onClose}>
      <DialogContent className="max-w-4xl max-h-[80vh] overflow-hidden flex flex-col">
        <DialogHeader className="flex-shrink-0">
          <DialogTitle className="flex items-center gap-2">
            <GitCommit className="h-5 w-5" />
            Commit Details
          </DialogTitle>
        </DialogHeader>

        <CommitDetailsContent commitHash={commitHash} />
      </DialogContent>
    </Dialog>
  );
}
```

- [ ] **Step 2: Commit**

```bash
git add src/components/repository/CommitDetailsDialog.tsx
git commit -m "refactor: simplify CommitDetailsDialog to use CommitDetailsContent

Dialog now wraps the extracted shared content component.

Co-Authored-By: Claude Opus 4.6 <noreply@anthropic.com>
"
```

---

### Task 3: Create CommitDetailsPage component

**Files:**
- Create: `src/pages/CommitDetailsPage.tsx`

- [ ] **Step 1: Create pages directory and CommitDetailsPage component**

```tsx
import { useParams, useNavigate } from 'react-router-dom';
import { ArrowLeft } from 'lucide-react';
import { Button } from '@/components/ui/button';
import CommitDetailsContent from '@/components/repository/CommitDetailsContent';

export default function CommitDetailsPage() {
  const { repoId, commitHash } = useParams<{ repoId: string; commitHash: string }>();
  const navigate = useNavigate();

  const handleBack = () => {
    navigate(`/repo/${repoId}`);
  };

  return (
    <div className="h-screen flex flex-col bg-background">
      {/* Header with back button */}
      <div className="flex items-center gap-4 p-4 border-b">
        <Button
          variant="ghost"
          size="icon"
          onClick={handleBack}
          aria-label="Back to commit history"
        >
          <ArrowLeft className="h-5 w-5" />
        </Button>
        <h1 className="text-lg font-semibold">Commit Details</h1>
      </div>

      {/* Commit content */}
      <CommitDetailsContent commitHash={commitHash || null} />
    </div>
  );
}
```

- [ ] **Step 2: Commit**

```bash
git add src/pages/CommitDetailsPage.tsx
git commit -m "feat: create CommitDetailsPage component

New page component for viewing commit details at
/repo/:repoId/commit/:commitHash route.

Co-Authored-By: Claude Opus 4.6 <noreply@anthropic.com>
"
```

---

### Task 4: Add route to App.tsx

**Files:**
- Modify: `src/App.tsx`

- [ ] **Step 1: Add import and route for CommitDetailsPage**

Update the file to:

```tsx
import { Routes, Route } from 'react-router-dom'
import AppLayout from './components/layout/AppLayout'
import LandingPage from './components/layout/LandingPage'
import ChooseRepositoryDialog from './components/repository/ChooseRepositoryDialog'
import CommitDetailsPage from './pages/CommitDetailsPage'

function App() {
  return (
    <div className="h-screen flex flex-col">
      <Routes>
        <Route path="/" element={<LandingPage />} />
        <Route path="/repo/:repoId" element={<AppLayout />} />
        <Route path="/repo/:repoId/commit/:commitHash" element={<CommitDetailsPage />} />
      </Routes>
      <ChooseRepositoryDialog />
    </div>
  )
}

export default App
```

- [ ] **Step 2: Commit**

```bash
git add src/App.tsx
git commit -m "feat: add commit details page route

Route: /repo/:repoId/commit/:commitHash

Co-Authored-By: Claude Opus 4.6 <noreply@anthropic.com>
"
```

---

### Task 5: Update CommitHistory to navigate instead of opening modal

**Files:**
- Modify: `src/components/repository/CommitHistory.tsx`

- [ ] **Step 1: Add useNavigate import and remove modal state**

At the top of the file, add:
```tsx
import { useNavigate } from 'react-router-dom';
```

Replace the state and dialog-related code:
```tsx
// REMOVE these lines:
// import CommitDetailsDialog from './CommitDetailsDialog';
// const [selectedCommitHash, setSelectedCommitHash] = useState<string | null>(null);

// ADD this line inside the component:
const navigate = useNavigate();
```

- [ ] **Step 2: Update setSelectedCommitHash calls to navigate**

Replace all occurrences of:
```tsx
setSelectedCommitHash(commit.hash)
```

With:
```tsx
navigate(`/repo/${currentRepository?.id}/commit/${commit.hash}`)
```

This appears in three places:
1. `onActivate` callback (line ~38)
2. Card onClick handler (line ~159)
3. View button onClick handler (line ~201)

- [ ] **Step 3: Remove CommitDetailsDialog render**

Remove these lines from the end of the component:
```tsx
<CommitDetailsDialog
    commitHash={selectedCommitHash}
    isOpen={!!selectedCommitHash}
    onClose={() => setSelectedCommitHash(null)}
/>
```

- [ ] **Step 4: Commit**

```bash
git add src/components/repository/CommitHistory.tsx
git commit -m "feat: navigate to commit page instead of opening modal

- Replace modal state with react-router navigation
- Clicking commit navigates to /repo/:repoId/commit/:hash
- Remove CommitDetailsDialog usage

Co-Authored-By: Claude Opus 4.6 <noreply@anthropic.com>
"
```

---

## Verification

After all tasks complete, verify:

- [ ] Run the dev server and navigate to a repository
- [ ] Click on a commit card - should navigate to commit details page
- [ ] URL should be `/repo/:repoId/commit/:commitHash`
- [ ] Back button returns to commit history
- [ ] Page refresh preserves commit view
- [ ] Vim navigation (Enter key) navigates to commit page
- [ ] Dialog still works if used elsewhere (regression check)
- [ ] Inline diff viewer works in page context
