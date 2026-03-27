# Repository Status Normalization: Strict Staged/Modified/Untracked Separation

**Date:** 2026-03-27
**Status:** Approved

## Problem

In change view, untracked files (`?`) can appear under **Staged Changes**. This indicates status normalization drift in the backend contract. The frontend currently renders whatever the API returns, so incorrect classification in API status data leaks directly into grouped UI sections.

## Goal

Enforce a strict backend status contract so untracked files never appear in `staged`, and each status bucket has one clear responsibility.

## Normalization Contract

- `staged`: index/staging changes for tracked content only.
  Allowed statuses: `A`, `M`, `D`, `R`, `C`.
  Any status outside this allowlist is omitted from `staged`.
- `modified`: worktree tracked changes only. Never include `?`.
- `untracked`: only file paths where worktree status is untracked (`?`) and not ignored by `.gitignore`.
- `conflicts`: unchanged from current behavior.

This preserves valid dual-state tracked files (for example, partially staged files can appear in both `staged` and `modified`) while keeping untracked paths isolated to `untracked`.

## Implementation Design

Update `GetRepositoryStatus` in `server/internal/git/service.go` to classify staging and worktree states independently with explicit guards:

- Add to `staged` only when:
  - `fileStatus.Staging` maps to one of `A`, `M`, `D`, `R`, `C`
  - otherwise omit from `staged` (no error)
- Add to `modified` only when:
  - `fileStatus.Worktree != git.Unmodified`
  - `fileStatus.Worktree != git.Untracked`
- Add to `untracked` only when:
  - `fileStatus.Worktree == git.Untracked`
  - and `gitignore.IsIgnored(file, false)` is false

No API schema changes are required. Sorting behavior remains unchanged.

## Data Flow Impact

- API output remains:
  - `staged: FileChange[]`
  - `modified: FileChange[]`
  - `untracked: string[]`
- Frontend grouping in `WorkingDirectoryChanges` continues to consume these arrays without UI-only patches.
- This fix applies consistently to all consumers of repository status, not only change view.

## Error Handling

No new API error paths are introduced. Classification is deterministic and tolerant:
- Unknown/non-allowlisted staging statuses are skipped in `staged`.
- Status retrieval failures (open repo, read worktree status, etc.) remain unchanged.

## Test Strategy

Add backend assertions in `server/internal/git/service_test.go` to lock contract behavior:

1. A newly created file appears in `Untracked` before staging.
2. `Staged` must not contain entries with `Status == "?"`.
3. `Staged` only contains allowlisted statuses (`A`, `M`, `D`, `R`, `C`).
4. An untracked path must not appear in `Staged`.
5. Existing tracked file modifications still appear in `Modified`.
6. A partially staged tracked file can appear in both `Staged` and `Modified`.

Prefer membership/contract assertions over brittle ordering-specific checks.

## Trade-offs

- Slightly more explicit branching in status assembly code.
- No frontend fallback filter is added; contract correctness is enforced at API layer.

## Out of Scope

- Conflict detection expansion.
- UI grouping refactors.
- Status schema changes.
