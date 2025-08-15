# Repository Components

This directory contains UI components for browsing and managing Git repositories in the web client.

## Overview
- **RepositoryPanel**: Switches between file browser, commit history, commit tree, branch management, working directory status, and settings views.
- **RepositoryList**: Lists available repositories and lets users create or import new ones.
- **ChooseRepositoryDialog**: Browse the filesystem to select and import an existing Git repository.
- **WorkingDirectoryChanges**: Shows staged, modified, untracked, and conflicted files with options to stage, unstage, view diffs, and commit.
- **CommitDialog**: Creates commits for staged files and captures commit message and optional author details.
- **CommitHistory** & **CommitDetailsDialog**: Display commit logs and inspect individual commit metadata and changes.
- **CommitTree**: Renders a visual graph of commit relationships.
- **BranchList** & **CreateBranchDialog**: Manage branches, including creation, switching, and deletion.
- **FolderSelectionDialog**: Placeholder component for choosing directories (pending UI implementation).

All components rely on Jotai atoms for shared state and query hooks for server communication.
