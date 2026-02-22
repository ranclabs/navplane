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

	"navplane/internal/config"
	"navplane/internal/database"
	"navplane/internal/handler"
	"navplane/internal/org"
	"navplane/internal/provider"
	"navplane/internal/providerkey"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("failed to load configuration: %v", err)
	}

	// Connect to database
	db, err := database.Connect(cfg.Database)
	if err != nil {
		log.Fatalf("failed to connect to database: %v", err)
	}
	defer func() {
		if err := db.Close(); err != nil {
			log.Printf("error closing database connection: %v", err)
		}
	}()
	log.Println("database connection established")

	// Run migrations
	migrationsPath := getMigrationsPath()
	if err := db.MigrateUp(migrationsPath); err != nil {
		log.Fatalf("failed to run migrations: %v", err)
	}
	version, dirty, err := db.MigrateVersion(migrationsPath)
	if err != nil {
		log.Printf("WARNING: failed to get migration version: %v", err)
	} else if dirty {
		log.Printf("WARNING: database is in dirty state at version %d - a previous migration failed and manual intervention is required", version)
	} else {
		log.Printf("database migrations complete (version: %d)", version)
	}

	// Initialize org manager
	orgDatastore := org.NewDatastore(db.DB)
	orgManager := org.NewManager(orgDatastore)

	// Initialize provider registry
	providerRegistry := provider.NewRegistry()
	log.Printf("registered providers: %v", providerRegistry.Names())

	// Initialize encryption and provider key manager
	encryptor, err := providerkey.NewEncryptor(cfg.Encryption.Key)
	if err != nil {
		log.Fatalf("failed to initialize encryptor: %v", err)
	}

	providerKeyDatastore := providerkey.NewDatastore(db.DB)
	providerKeyManager := providerkey.NewManager(providerKeyDatastore, encryptor, providerRegistry)

	// Set up routes with dependencies
	deps := &handler.Deps{
		Config:             cfg,
		OrgManager:         orgManager,
		ProviderRegistry:   providerRegistry,
		ProviderKeyManager: providerKeyManager,
	}

	mux := http.NewServeMux()
	handler.RegisterRoutes(mux, deps)

	server := &http.Server{
		Addr:    ":" + cfg.Port,
		Handler: mux,
	}

	// Channel to listen for shutdown signals
	shutdown := make(chan os.Signal, 1)
	signal.Notify(shutdown, syscall.SIGINT, syscall.SIGTERM)

	// Channel to signal server errors
	serverErr := make(chan error, 1)

	// Start server in a goroutine
	go func() {
		log.Printf("NavPlane server starting on :%s (env: %s)", cfg.Port, cfg.Environment)
		serverErr <- server.ListenAndServe()
	}()

	// Block until we receive a shutdown signal or server error
	select {
	case err := <-serverErr:
		if err != nil && err != http.ErrServerClosed {
			log.Fatalf("server error: %v", err)
		}
	case sig := <-shutdown:
		log.Printf("received signal %v, initiating graceful shutdown...", sig)

		// Create a context with timeout for the shutdown
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		// Attempt graceful shutdown
		log.Println("waiting for in-flight requests to complete...")
		if err := server.Shutdown(ctx); err != nil {
			log.Printf("graceful shutdown failed: %v, forcing shutdown", err)
			if err := server.Close(); err != nil {
				log.Fatalf("forced shutdown failed: %v", err)
			}
		}

		log.Println("server shutdown complete")
	}
}

// getMigrationsPath returns the path to the migrations directory.
// It checks for the migrations folder relative to the executable or working directory.
func getMigrationsPath() string {
	// Check if MIGRATIONS_PATH env var is set
	if path := os.Getenv("MIGRATIONS_PATH"); path != "" {
		return path
	}

	// Try relative to working directory (for local development)
	if _, err := os.Stat("migrations"); err == nil {
		absPath, _ := filepath.Abs("migrations")
		return absPath
	}

	// Try relative to executable (for Docker)
	execPath, err := os.Executable()
	if err == nil {
		execDir := filepath.Dir(execPath)
		migrationsPath := filepath.Join(execDir, "migrations")
		if _, err := os.Stat(migrationsPath); err == nil {
			return migrationsPath
		}
	}

	// Default fallback
	return "/app/migrations"
}
