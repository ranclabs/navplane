package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"navplane/internal/config"
	"navplane/internal/handler"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("failed to load configuration: %v", err)
	}

	mux := http.NewServeMux()
	handler.RegisterRoutes(mux, cfg)

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
