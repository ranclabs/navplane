package handler

import (
	"net/http"

	"navplane/internal/config"
)

// RegisterRoutes registers all HTTP routes with the provided mux.
// The config is passed to handlers that need access to provider configuration.
func RegisterRoutes(mux *http.ServeMux, cfg *config.Config) {
	// Health and status endpoints (no auth required)
	mux.HandleFunc("GET /health", HealthCheck)
	mux.HandleFunc("GET /api/v1/status", statusHandler(cfg))

	// OpenAI-compatible API endpoints
	// Uses POST method pattern for proper 405 handling by ServeMux
	mux.HandleFunc("POST /v1/chat/completions", NewChatCompletionsHandler(cfg))

	// Also register without method to return proper 405 for other methods
	// This ensures GET, PUT, DELETE, etc. get a proper error response
	mux.HandleFunc("/v1/chat/completions", methodNotAllowedHandler("POST"))
}

func methodNotAllowedHandler(allowedMethods string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Allow", allowedMethods)
		writeProxyError(w, http.StatusMethodNotAllowed, "method not allowed", "invalid_request_error")
	}
}
