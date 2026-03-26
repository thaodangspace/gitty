package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"gitweb/server/internal/api"
	"gitweb/server/internal/auth"
	"gitweb/server/internal/config"
	"gitweb/server/internal/registry"

	"github.com/go-chi/chi/v5"
)

func main() {
	homeDir, _ := os.UserHomeDir()

	dataPath := os.Getenv("GITTY_DATA_PATH")
	if dataPath == "" {
		dataPath = filepath.Join(homeDir, ".gitty", "repositories")
	}

	if err := os.MkdirAll(dataPath, 0o755); err != nil {
		log.Fatalf("Failed to create data directory: %v", err)
	}

	masterPassword := os.Getenv("GITTY_MASTER_PASSWORD")

	cfg, err := config.Load()
	if err != nil {
		log.Printf("Warning: Failed to load config: %v", err)
		cfg = &config.Config{}
	}

	// Set master password from environment variable if provided (takes precedence over config)
	if masterPassword != "" {
		cfg.MasterPassword = &masterPassword
	}

	// Initialize registry
	registryPath := filepath.Join(homeDir, ".config", "gitty", "repository.json")
	reg, err := registry.New(registryPath)
	if err != nil {
		log.Fatalf("Failed to load repository registry: %v", err)
	}

	// Warn about legacy data directory (only if ~/.gitweb exists and ~/.gitty does not)
	legacyPath := filepath.Join(homeDir, ".gitweb")
	newPath := filepath.Join(homeDir, ".gitty")
	if _, err := os.Stat(legacyPath); err == nil {
		if _, err := os.Stat(newPath); os.IsNotExist(err) {
			log.Printf("Warning: Legacy directory %s found but %s does not exist. Repositories stored there are no longer auto-discovered. Use the import API to re-add them.", legacyPath, newPath)
		}
	}

	apiRouter := api.NewRouter(dataPath, cfg, reg)

	r := chi.NewRouter()

	r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status":"ok","service":"gittyd"}`))
	})

	r.Group(func(r chi.Router) {
		r.Use(auth.PasswordGate(masterPassword))
		r.Mount("/", apiRouter)
	})

	port := os.Getenv("PORT")
	if port == "" {
		port = "8083"
	}

	srv := &http.Server{
		Addr:    ":" + port,
		Handler: r,
	}

	go func() {
		log.Printf("Starting Gitty daemon on :%s", port)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Server failed to start: %v", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	log.Println("Shutting down server...")

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		log.Fatalf("Server forced to shutdown: %v", err)
	}

	log.Println("Server exited")
}
