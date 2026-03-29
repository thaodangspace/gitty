package main

import (
	"context"
	"fmt"
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
	"gitweb/server/internal/resources"

	"github.com/go-chi/chi/v5"
)

func main() {
	appCtx, appCancel := context.WithCancel(context.Background())
	defer appCancel()

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
		log.Fatalf("Failed to load config: %v", err)
	}

	// Set master password from environment variable if provided (takes precedence over config)
	if masterPassword != "" {
		cfg.MasterPassword = &masterPassword
	}

	// Fail fast if masterPassword is missing after env + config merge
	if !cfg.HasMasterPassword() {
		log.Fatal("GITTY_MASTER_PASSWORD environment variable or config file masterPassword is required")
	}

	runtimeCaps, err := resources.RuntimeCapsFromAppConfig(cfg)
	if err != nil {
		log.Fatalf("Invalid resource governor config: %v", err)
	}
	if runtimeCaps.Enabled {
		caps, err := resources.ApplyRuntimeCaps(runtimeCaps)
		if err != nil {
			log.Fatalf("Invalid resource governor config: %v", err)
		}
		log.Printf("Resource caps applied: memory=%d gomaxprocs=%d", caps.MemoryLimitBytes, caps.GOMAXPROCS)
	} else {
		log.Printf("Resource caps disabled; skipping runtime application")
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

	// Initialize auth components
	tokenStorePath := filepath.Join(homeDir, ".config", "gitty", "auth-tokens.json")
	if err := os.MkdirAll(filepath.Dir(tokenStorePath), 0o700); err != nil {
		log.Fatalf("Failed to create auth directory: %v", err)
	}

	tokenStore, err := auth.NewTokenStore(tokenStorePath)
	if err != nil {
		log.Fatalf("Failed to initialize token store: %v", err)
	}

	pairingManager := auth.NewPairingManager(auth.DefaultPairSessionTTL)

	// Create initial pairing session and print QR payload
	session, err := pairingManager.CreateSession()
	if err != nil {
		log.Fatalf("Failed to create pairing session: %v", err)
	}
	qrPayload := fmt.Sprintf("gitty-pair://session?id=%s", session.SessionID)
	log.Printf("Pairing session: %s", qrPayload)

	apiRouter := api.NewRouter(appCtx, dataPath, cfg, reg, pairingManager, tokenStore)

	r := chi.NewRouter()

	// Health endpoint is mounted in apiRouter, but we add a simple one here too
	r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status":"ok","service":"gittyd"}`))
	})

	r.Mount("/", apiRouter)

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
	appCancel()

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		log.Fatalf("Server forced to shutdown: %v", err)
	}

	log.Println("Server exited")
}
