package handler

import (
	"net/http"
	"navplane/internal/config"
)

// RegisterRoutes registers all HTTP routes with the provided mux.
// The config is passed to handlers that need access to provider configuration.
func RegisterRoutes(mux *http.ServeMux, cfg *config.Config) {
	// Health and status endpoints
	mux.HandleFunc("GET /health", HealthCheck)
	mux.HandleFunc("GET /api/v1/status", statusHandler(cfg))

	// OpenAI-compatible API endpoints
	// Note: We register without method prefix to handle 405 in the handler
	mux.HandleFunc("/v1/chat/completions", ChatCompletions)
}
