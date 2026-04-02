// @title           Gittyd API
// @version         1.0
// @description     GitWeb's backend API for managing Git repositories and browsing the local filesystem.
// @description     This API provides endpoints for repository management, branch/commit operations, file management, and remote sync.
// @termsOfService  http://swagger.io/terms/

// @contact.name   API Support
// @contact.email  support@gitty.local

// @license.name   MIT
// @license.url    https://opensource.org/licenses/MIT

// @host           localhost:8083
// @BasePath       /

// @securityDefinitions.apikey BearerAuth
// @in header
// @name Authorization
// @description Enter "Bearer <token>" where <token> is obtained from the /api/auth/pair/exchange endpoint

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
	"gitweb/server/internal/api/handlers"
	"gitweb/server/internal/auth"
	"gitweb/server/internal/config"
	"gitweb/server/internal/registry"
	"gitweb/server/internal/resources"

	"github.com/go-chi/chi/v5"
	"github.com/mdp/qrterminal/v3"

	_ "gitweb/server/docs" // swagger docs
	httpSwagger "github.com/swaggo/http-swagger"
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

	// Determine port early for QR payload
	port := os.Getenv("PORT")
	if port == "" {
		port = "8083"
	}

	// Create initial pairing session and print QR payload
	session, err := pairingManager.CreateSession()
	if err != nil {
		log.Fatalf("Failed to create pairing session: %v", err)
	}

	// Build QR payload with network-reachable URL
	baseURL := os.Getenv("GITTY_BASE_URL")
	if baseURL == "" {
		// Try to detect local IP
		hostname, _ := os.Hostname()
		baseURL = fmt.Sprintf("http://%s:%s", hostname, port)
	}
	qrPayload := fmt.Sprintf(`{"baseUrl":"%s","sessionId":"%s","expiresAt":"%s"}`,
		baseURL,
		session.SessionID,
		session.ExpiresAt.Format(time.RFC3339))

	// Render ASCII QR code to stderr (so it doesn't interfere with regular logs)
	log.Println("Scan this QR code to pair your device:")
	qrterminal.GenerateHalfBlock(qrPayload, qrterminal.L, os.Stderr)

	// Also print plain-text fallback
	log.Printf("Plain-text pairing payload: %s", qrPayload)

	// Initialize auth handler with the current session ID
	authHandler := handlers.NewAuthHandlerWithSession(pairingManager, tokenStore, *cfg.MasterPassword, session.SessionID)

	apiRouter := api.NewRouter(appCtx, dataPath, cfg, reg, pairingManager, tokenStore, authHandler)

	r := chi.NewRouter()

	// Swagger UI will be mounted here in Task 3
	_ = httpSwagger.WrapHandler

	// Health endpoint is mounted in apiRouter, but we add a simple one here too
	r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status":"ok","service":"gittyd"}`))
	})

	r.Mount("/", apiRouter)

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
