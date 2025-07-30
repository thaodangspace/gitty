package api

import (
	"gitweb/server/internal/api/handlers"
	"gitweb/server/internal/api/middleware"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/cors"
)

func NewRouter(dataPath string) *chi.Mux {
	r := chi.NewRouter()

	r.Use(cors.Handler(cors.Options{
		AllowedOrigins:   []string{"http://localhost:5173", "http://localhost:3000", "http://100.81.122.10:5173"},
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type", "X-CSRF-Token"},
		ExposedHeaders:   []string{"Link"},
		AllowCredentials: true,
		MaxAge:           300,
	}))

	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)

	repoHandler := handlers.NewRepositoryHandler(dataPath)
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
				
				r.Post("/commit", repoHandler.CreateCommit)
				r.Post("/branches", repoHandler.CreateBranch)
				r.Put("/branches/{branch}", repoHandler.SwitchBranch)
				r.Delete("/branches/{branch}", repoHandler.DeleteBranch)
				
				r.Get("/files", repoHandler.GetFileTree)
				r.Get("/files/*", repoHandler.GetFileContent)
				r.Put("/files/*", repoHandler.SaveFileContent)
				
				r.Get("/diff/*", repoHandler.GetFileDiff)
				
				r.Post("/stage/*", repoHandler.StageFile)
				r.Delete("/stage/*", repoHandler.UnstageFile)
				
				r.Post("/push", repoHandler.Push)
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
