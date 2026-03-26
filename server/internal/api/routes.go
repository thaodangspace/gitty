package api

import (
	"gitweb/server/internal/api/handlers"
	"gitweb/server/internal/api/middleware"
	"gitweb/server/internal/config"
	"gitweb/server/internal/registry"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/cors"
)

func NewRouter(dataPath string, cfg *config.Config, reg *registry.Registry) *chi.Mux {
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

	repoHandler := handlers.NewRepositoryHandler(dataPath, cfg, reg)
	fsHandler := handlers.NewFilesystemHandler(true) // Restrict to user home directory

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

				r.Post("/commit", repoHandler.CreateCommit)
				r.Post("/generate-commit-message", repoHandler.GenerateCommitMessage)
				r.Post("/branches", repoHandler.CreateBranch)
				r.Put("/branches/{branch}", repoHandler.SwitchBranch)
				r.Delete("/branches/{branch}", repoHandler.DeleteBranch)

				r.Get("/files", repoHandler.GetFileTree)
				r.Get("/files/*", repoHandler.GetFileContent)
				r.Put("/files/*", repoHandler.SaveFileContent)

				r.Get("/diff/*", repoHandler.GetFileDiff)
				r.Get("/diff/tokenized/*", repoHandler.HandleTokenizedFileDiff)
				r.Get("/diff/commit/tokenized", repoHandler.HandleTokenizedCommitDiff)
			r.Get("/diff/commit/{hash}/files/*", repoHandler.HandleCommitFileDiff)

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

	return r
}
