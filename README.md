# GitWeb: A Web-Based Git Client

GitWeb is a modern, web-based Git client designed for developers who need a visual and intuitive interface to manage their repositories. It's built with a Go backend and a React frontend, offering a fast and responsive experience. This tool is particularly useful for those who prefer a graphical interface over the command line and for managing projects on mobile devices.

## Features

-   **Repository Management**: Clone existing repositories or initialize new ones.
-   **Visual Diff Viewer**: See the differences between commits, branches, and your working directory.
-   **Commit History**: Browse the commit history of your projects.
-   **Branch Management**: Create, switch, and manage branches.
-   **Staging Area**: Stage and unstage changes with ease.
-   **Mobile-Friendly**: A responsive design that works on mobile devices.
-   **Real-time Updates**: Changes are reflected in real-time.

## Technology Stack

-   **Backend**: Go with the `chi` router
-   **Frontend**: React, Vite, Tailwind CSS, shadcn/ui, Jotai, and React Query
-   **Real-time Communication**: WebSockets
-   **Git Operations**: `go-git` library

## Getting Started

To get started with GitWeb, follow these instructions to set up the project on your local machine.

### Prerequisites

-   Go (version 1.18 or higher)
-   Node.js (version 16 or higher)
-   npm

### Installation & Running

1.  **Clone the repository:**

    ```bash
    git clone https://github.com/your-username/gitweb.git
    cd gitweb
    ```

2.  **Run the backend server:**

    Open a terminal and navigate to the `server` directory. Then, run the following command to start the Go server:

    ```bash
    cd server
    go run ./cmd/gitweb
    ```

    The backend server will start on `http://localhost:8080`.

3.  **Run the frontend application:**

    In a new terminal, navigate to the `www` directory and install the dependencies:

    ```bash
    cd www
    npm install
    ```

    Then, start the frontend development server:

    ```bash
    npm run dev
    ```

    The frontend will be available at `http://localhost:5173`.

## Architecture

GitWeb is composed of a Go backend that serves a REST API and a React single-page application (SPA) for the frontend. The backend uses the `go-git` library to perform Git operations on the local file system. WebSockets are used for real-time communication between the frontend and backend.

For more details, see the [ARCHITECTURE.md](ARCHITECTURE.md) file.

## Project Status

This project is currently in **Phase 5: Advanced Features**. Most of the core features are implemented. For a detailed list of completed, in-progress, and planned features, please refer to the [CHECKLIST.md](CHECKLIST.md) file.

## Contributing

Contributions are welcome! If you'd like to contribute, please fork the repository and create a pull request. For major changes, please open an issue first to discuss what you would like to change.

## License

This project is licensed under the MIT License - see the `LICENSE` file for details.
