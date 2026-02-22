package handler

import (
	"net/http"

	"navplane/internal/config"
	"navplane/internal/middleware"
	"navplane/internal/org"
)

// Deps contains dependencies for route handlers.
type Deps struct {
	Config     *config.Config
	OrgManager *org.Manager
}

// RegisterRoutes registers all HTTP routes with the provided mux.
func RegisterRoutes(mux *http.ServeMux, deps *Deps) {
	// Health and status endpoints (no auth required)
	mux.HandleFunc("GET /health", HealthCheck)
	mux.HandleFunc("GET /api/v1/status", statusHandler(deps.Config))

	// Auth middleware for protected routes
	authMiddleware := middleware.Auth(deps.OrgManager)

	// OpenAI-compatible API endpoints (auth required)
	chatHandler := NewChatCompletionsHandler(deps.Config)
	mux.Handle("POST /v1/chat/completions", authMiddleware(http.HandlerFunc(chatHandler)))

	// Return proper 405 for other methods on protected endpoints
	mux.HandleFunc("/v1/chat/completions", methodNotAllowedHandler("POST"))
}

func methodNotAllowedHandler(allowedMethods string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Allow", allowedMethods)
		writeProxyError(w, http.StatusMethodNotAllowed, "method not allowed", "invalid_request_error")
	}
}
