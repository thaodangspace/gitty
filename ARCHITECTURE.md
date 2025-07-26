# GitWeb - Web-Based Git Client

## System Architecture

### Technology Stack
- **Backend**: Go + go-chi (REST API)
- **Frontend**: React.js + Tailwind CSS + Shadcn/ui + React Query + Jotai
- **Real-time**: WebSocket connections
- **Git Operations**: go-git library or git CLI wrapper

### Architecture Overview
```
┌─────────────────┐    ┌──────────────────┐    ┌─────────────────┐
│   React Client  │◄──►│   Go API Server  │◄──►│  Git Repositories│
│  (Tailwind +    │    │    (go-chi)      │    │  (File System)  │
│   Shadcn/ui)    │    │                  │    │                 │
└─────────────────┘    └──────────────────┘    └─────────────────┘
         ▲                        ▲
         │                        │
         └────── WebSocket ───────┘
```

## Backend API Structure

### Core Endpoints
```
Repository Management:
- GET    /api/repos              # List repositories
- POST   /api/repos              # Add repository (clone/init)
- GET    /api/repos/{id}         # Repository details
- DELETE /api/repos/{id}         # Remove repository

File Operations:
- GET    /api/repos/{id}/files          # File tree
- GET    /api/repos/{id}/files/{path}   # File content
- PUT    /api/repos/{id}/files/{path}   # Save file

Git Operations:
- GET    /api/repos/{id}/status         # Git status
- GET    /api/repos/{id}/commits        # Commit history
- GET    /api/repos/{id}/branches       # Branch list
- POST   /api/repos/{id}/commit         # Create commit
- POST   /api/repos/{id}/branches       # Create branch
- PUT    /api/repos/{id}/branches/{name} # Switch branch
- POST   /api/repos/{id}/push           # Push to remote
- POST   /api/repos/{id}/pull           # Pull from remote

Diff/Compare:
- GET    /api/repos/{id}/diff           # Working directory diff
- GET    /api/repos/{id}/diff/{commit1}/{commit2} # Compare commits

Real-time:
- WS     /ws                           # WebSocket endpoint
```

## Frontend Component Architecture

### Layout Components
- `AppLayout` - Main application container
- `Sidebar` - Repository list and navigation
- `Header` - Toolbar and status indicators
- `StatusBar` - Bottom status information

### Feature Components
- `RepositoryList` - Repository management sidebar
- `RepositoryPanel` - Main repository view
- `FileTree` - File browser with tree structure
- `FileViewer` - Code editor and file content
- `CommitHistory` - Git log visualization
- `DiffViewer` - File difference display
- `StagingArea` - Changes staging interface
- `CommitDialog` - Commit creation form
- `BranchManager` - Branch operations panel

### State Management
- **Jotai**: Atomic state management for global app state
  - Repository atoms (current repo, branches, status)
  - UI atoms (sidebar state, selected files, active views)
  - Settings atoms (user preferences, themes)
- **React Query**: Server state management and caching
  - Repository data fetching and mutations
  - File content caching and synchronization
  - Background refetching and optimistic updates
- **Context API**: Component-specific state (themes, user sessions)
- **WebSocket**: Real-time updates integration with both Jotai and React Query

## Implementation Checklist

### Phase 1: Project Setup
- [ ] Initialize Go module with go-chi
- [ ] Setup React project with Vite
- [ ] Configure Tailwind CSS and Shadcn/ui
- [ ] Setup React Query and Jotai
- [ ] Setup project structure and basic routing
- [ ] Implement CORS and basic middleware

### Phase 2: Core Backend
- [ ] Repository management endpoints
- [ ] Git wrapper service (using go-git)
- [ ] File system operations
- [ ] Basic authentication/session handling
- [ ] WebSocket connection manager

### Phase 3: Core Frontend
- [ ] Main layout and navigation
- [ ] Repository list component
- [ ] File tree browser
- [ ] Basic file viewer
- [ ] API client setup with React Query integration
- [ ] Jotai atoms for global state management

### Phase 4: Git Operations
- [ ] Commit history display
- [ ] Branch management
- [ ] Staging and committing
- [ ] Diff viewer implementation
- [ ] Push/pull operations

### Phase 5: Advanced Features
- [ ] Merge conflict resolution
- [ ] Advanced diff visualization
- [ ] Search functionality
- [ ] Settings and preferences
- [ ] SSH key management

### Phase 6: Polish & Deployment
- [ ] Error handling and user feedback
- [ ] Performance optimization
- [ ] Testing (unit + integration)
- [ ] Docker containerization
- [ ] Production deployment setup

## Project Structure

### Backend (Go)
```
backend/
├── cmd/
│   └── server/
│       └── main.go
├── internal/
│   ├── api/
│   │   ├── handlers/
│   │   ├── middleware/
│   │   └── routes.go
│   ├── git/
│   │   └── service.go
│   ├── models/
│   └── websocket/
├── pkg/
└── go.mod
```

### Frontend (React)
```
frontend/
├── src/
│   ├── components/
│   │   ├── layout/
│   │   ├── repository/
│   │   ├── file/
│   │   └── ui/
│   ├── hooks/
│   ├── contexts/
│   ├── store/
│   │   ├── atoms/
│   │   └── queries/
│   ├── api/
│   ├── utils/
│   └── App.tsx
├── public/
├── package.json
└── vite.config.ts
```

## Key Features

### Core Git Operations
- Repository cloning and initialization
- File editing and saving
- Staging and committing changes
- Branch creation and switching
- Merge operations
- Push/pull from remotes

### User Interface
- Split-pane layout with file tree and content
- Syntax highlighting for code files
- Visual diff viewer
- Interactive staging area
- Real-time status updates

### Advanced Features
- Merge conflict resolution interface
- Search across repository files
- Settings and preferences management
- SSH key management for remote operations

This architecture provides a solid foundation for a full-featured web-based Git client similar to Fork/GitKraken.