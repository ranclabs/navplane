package handler

import (
	"net/http"

	"navplane/internal/auth"
	"navplane/internal/config"
	"navplane/internal/middleware"
)

// RegisterRoutes registers all HTTP routes with the provided mux.
// The config is passed to handlers that need access to provider configuration.
// The authStore is used for authenticating requests to protected endpoints.
func RegisterRoutes(mux *http.ServeMux, cfg *config.Config, authStore auth.AuthStore) {
	// Health and status endpoints (public, no auth required)
	mux.HandleFunc("GET /health", HealthCheck)
	mux.HandleFunc("GET /api/v1/status", statusHandler(cfg))

	// Auth middleware for protected endpoints
	requireAuth := middleware.RequireAuth(authStore)

	// OpenAI-compatible API endpoints (protected, require auth)
	// Note: We register without method prefix to handle 405 in the handler
	mux.Handle("/v1/chat/completions", requireAuth(chatCompletionsHandler(cfg)))
}
