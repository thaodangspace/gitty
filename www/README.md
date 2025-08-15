# GitWeb Frontend

GitWeb's frontend is a React + TypeScript single page application built with [Vite](https://vitejs.dev). It provides a mobile-first interface for interacting with the GitWeb backend and managing Git repositories visually.

## Features

- **Repository management** – create, import, delete, and switch between repositories
- **Branch operations** – list branches, create new ones and switch between them
- **Commit history** – browse commits and view commit details and diffs
- **File browser** – explore the repository tree, view files and see diffs for changed files
- **Responsive layout** – desktop sidebar and mobile drawer for navigation
- **Real-time updates** – repository status and data kept fresh via React Query

## Tech Stack

- React 19 & TypeScript
- Vite build tool
- Tailwind CSS & [shadcn/ui](https://ui.shadcn.com)
- [Jotai](https://jotai.org) for local state management
- [React Query](https://tanstack.com/query/latest) for server state
- React Router for routing

## Getting Started

1. **Install dependencies**
   ```bash
   npm install
   ```

2. **Start the development server**
   ```bash
   npm run dev
   ```
   The app will run at `http://localhost:5173` and expects the backend at `http://localhost:8080` by default. You can override the backend URL by setting `VITE_API_BASE`.

3. **Lint the project**
   ```bash
   npm run lint
   ```

4. **Create a production build**
   ```bash
   npm run build
   ```

## Project Structure

```
www/
  src/
    components/   # UI components (repository panels, file viewers, layout)
    hooks/        # custom hooks including API helpers
    lib/          # API client and utility helpers
    store/        # Jotai atoms and React Query keys
    types/        # shared TypeScript types
```

## Environment Variables

- `VITE_API_BASE` – URL of the GitWeb backend (defaults to `http://localhost:8080`)

## Available Scripts

- `npm run dev` – start Vite dev server with hot reload
- `npm run build` – type-check and create production build
- `npm run lint` – run ESLint over the codebase
- `npm run preview` – preview the production build locally

