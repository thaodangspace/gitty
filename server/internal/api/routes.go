package api

import (
	"context"
	"net/http"
	"gitweb/server/internal/api/handlers"
	"gitweb/server/internal/api/middleware"
	"gitweb/server/internal/auth"
	"gitweb/server/internal/config"
	"gitweb/server/internal/registry"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/cors"
)

func NewRouter(ctx context.Context, dataPath string, cfg *config.Config, reg *registry.Registry, pm *auth.PairingManager, ts *auth.TokenStore, authHandler *handlers.AuthHandler) *chi.Mux {
	r := chi.NewRouter()

	r.Use(cors.Handler(cors.Options{
		AllowedOrigins:   []string{"http://localhost:5176", "http://100.117.191.67:5176"},
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type", "X-CSRF-Token"},
		ExposedHeaders:   []string{"Link"},
		AllowCredentials: true,
		MaxAge:           300,
	}))

	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)

	// Health endpoint (public, no auth required)
	r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status":"ok","service":"gittyd"}`))
	})

	// Initialize handlers
	repoHandler := newRepoHandler(ctx, dataPath, cfg, reg)
	fsHandler := newFsHandler()

	// Auth handler is passed from main.go with session ID already set

	// Public auth routes (no bearer required)
	r.Route("/api/auth", func(r chi.Router) {
		// Pair exchange endpoint is public for initial device pairing
		r.Post("/pair/exchange", authHandler.PairExchange)
		// Pair session endpoint - returns current session info for QR scanning
		r.Get("/pair/session", authHandler.GetPairSession)
		// Local-only pairing for web frontend (localhost only)
		r.Post("/local/pair", authHandler.LocalPair)
	})

	// Protected routes (bearer required)
	r.Group(func(r chi.Router) {
		r.Use(auth.BearerGate(ts))

		// Device management routes (behind bearer)
		r.Route("/api/auth/devices", func(r chi.Router) {
			r.Get("/", authHandler.ListDevices)
			r.Delete("/{deviceId}", authHandler.RevokeDevice)
		})

		// Repository and filesystem routes
		r.Route("/api", func(r chi.Router) {
			r.Route("/repos", func(r chi.Router) {
				r.Get("/", repoHandler.ListRepositories)
				r.Post("/", repoHandler.CreateRepository)
				r.Post("/import", repoHandler.ImportRepository)

				r.Route("/{id}", func(r chi.Router) {
					r.Get("/", repoHandler.GetRepository)
					r.Delete("/", repoHandler.DeleteRepository)

					r.Get("/status", repoHandler.GetRepositoryStatus)
					r.Get("/commits", repoHandler.GetCommitHistory)
					r.Get("/commits/{hash}", repoHandler.GetCommitDetails)
					r.Get("/branches", repoHandler.GetBranches)
					r.Get("/config/git", repoHandler.GetGitConfig)
					r.Get("/settings", repoHandler.GetRepositorySettings)
					r.Put("/settings/identity", repoHandler.UpdateRepositorySettingsIdentity)
					r.Put("/settings/sync", repoHandler.UpdateRepositorySettingsSync)
					r.Put("/settings/commit", repoHandler.UpdateRepositorySettingsCommit)

					r.Post("/commit", repoHandler.CreateCommit)
					r.Post("/generate-commit-message", repoHandler.GenerateCommitMessage)
					r.Post("/branches", repoHandler.CreateBranch)
					r.Put("/branches/{branch}", repoHandler.SwitchBranch)
					r.Delete("/branches/{branch}", repoHandler.DeleteBranch)

					r.Get("/files", repoHandler.GetFileTree)
					r.Get("/files/*", repoHandler.GetFileContent)
					r.Put("/files/*", repoHandler.SaveFileContent)

					// Specific routes first (before /diff/*)
					r.Get("/diff/commit/{hash}/files/*", repoHandler.HandleCommitFileDiff)
					r.Get("/diff/commit/tokenized", repoHandler.HandleTokenizedCommitDiff)
					r.Get("/diff/tokenized/*", repoHandler.HandleTokenizedFileDiff)
					// General diff route last
					r.Get("/diff/*", repoHandler.GetFileDiff)

					r.Post("/stage/*", repoHandler.StageFile)
					r.Post("/stage-all", repoHandler.StageAllFiles)
					r.Delete("/stage/*", repoHandler.UnstageFile)

					r.Post("/push", repoHandler.Push)
					r.Post("/push/force", repoHandler.ForcePush)
					r.Post("/pull", repoHandler.Pull)
				})
			})

			r.Route("/filesystem", func(r chi.Router) {
				r.Get("/browse", fsHandler.BrowseDirectory)
				r.Get("/roots", fsHandler.GetVolumeRoots)
			})
		})
	})

	return r
}

func newRepoHandler(ctx context.Context, dataPath string, cfg *config.Config, reg *registry.Registry) *handlers.RepositoryHandler {
	repoHandler := handlers.NewRepositoryHandler(dataPath, cfg, reg)
	repoHandler.StartPressureMonitor(ctx)
	return repoHandler
}

func newFsHandler() *handlers.FilesystemHandler {
	return handlers.NewFilesystemHandler(true) // Restrict to user home directory
}
