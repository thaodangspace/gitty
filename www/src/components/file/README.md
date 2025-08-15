# File Browsing Components

This directory contains React components that power the repository file interface in the GitWeb frontend.

## Components

### FileTreeBrowser.tsx
- Fetches the repository's file tree via `apiClient.getFileTree` and constructs a nested structure.
- Allows expanding and collapsing directories and selects files using Jotai atoms.
- Designed to work on both desktop and mobile layouts.

### FileViewer.tsx
- Displays the currently selected file.
- Renders text files, images, or a placeholder for binary content.
- Provides basic actions such as view, edit for text files, and download.

### DiffViewer.tsx
- Retrieves a file's diff with `apiClient.getFileDiff`.
- Parses unified diff output and highlights additions, removals, and context lines with line numbers.
- Presented in a modal with loading and error states.

## Usage
These components are intended to be used together: `FileTreeBrowser` updates the selected file, `FileViewer` renders its contents, and `DiffViewer` can display changes for that file. They depend on shared Jotai atoms for repository and file selection and use React Query for data fetching.
