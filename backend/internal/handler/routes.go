package handler

import (
	"net/http"
	"navplane/internal/config"
)

// RegisterRoutes registers all HTTP routes with the provided mux.
// The config is passed to handlers that need access to provider configuration.
func RegisterRoutes(mux *http.ServeMux, cfg *config.Config) {
	mux.HandleFunc("GET /health", HealthCheck)
	mux.HandleFunc("GET /api/v1/status", statusHandler(cfg))
}
