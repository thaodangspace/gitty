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

	"github.com/go-chi/chi/v5"
)

func main() {
	dataPath := os.Getenv("GITWEB_DATA_PATH")
	if dataPath == "" {
		homeDir, _ := os.UserHomeDir()
		dataPath = filepath.Join(homeDir, ".gitweb", "repositories")
	}

	if err := os.MkdirAll(dataPath, 0o755); err != nil {
		log.Fatalf("Failed to create data directory: %v", err)
	}

	masterPassword := os.Getenv("GITTY_MASTER_PASSWORD")

	apiRouter := api.NewRouter(dataPath)

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
