package handler

import (
	"net/http"

	"navplane/internal/config"
	"navplane/internal/middleware"
	"navplane/internal/org"
	"navplane/internal/provider"
	"navplane/internal/providerkey"
)

// Deps contains dependencies for route handlers.
type Deps struct {
	Config             *config.Config
	OrgManager         *org.Manager
	ProviderRegistry   *provider.Registry
	ProviderKeyManager *providerkey.Manager
}

// RegisterRoutes registers all HTTP routes with the provided mux.
func RegisterRoutes(mux *http.ServeMux, deps *Deps) {
	// Health and status endpoints (no auth required)
	mux.HandleFunc("GET /health", HealthCheck)
	mux.HandleFunc("GET /api/v1/status", statusHandler(deps.Config))

	// Auth middleware for protected routes
	authMiddleware := middleware.Auth(deps.OrgManager)

	// OpenAI-compatible API endpoints (auth required)
	chatDeps := &ChatCompletionsDeps{
		ProviderRegistry:   deps.ProviderRegistry,
		ProviderKeyManager: deps.ProviderKeyManager,
	}
	chatHandler := NewChatCompletionsHandler(chatDeps)
	mux.Handle("POST /v1/chat/completions", authMiddleware(http.HandlerFunc(chatHandler)))

	// Return proper 405 for other methods on protected endpoints
	mux.HandleFunc("/v1/chat/completions", methodNotAllowedHandler("POST"))

	// Admin API endpoints
	// TODO: Add admin authentication (separate from org auth)
	registerAdminRoutes(mux, deps)
}

// registerAdminRoutes registers admin API endpoints.
// These endpoints are for dashboard/internal use only.
func registerAdminRoutes(mux *http.ServeMux, deps *Deps) {
	adminOrgs := NewAdminOrgsHandler(deps.OrgManager)

	// Organization management
	mux.HandleFunc("GET /admin/orgs", adminOrgs.List)
	mux.HandleFunc("POST /admin/orgs", adminOrgs.Create)
	mux.HandleFunc("GET /admin/orgs/{id}", adminOrgs.Get)
	mux.HandleFunc("PUT /admin/orgs/{id}", adminOrgs.Update)
	mux.HandleFunc("DELETE /admin/orgs/{id}", adminOrgs.Delete)

	// Kill switch - enable/disable org
	mux.HandleFunc("PUT /admin/orgs/{id}/enabled", adminOrgs.SetEnabled)

	// API key rotation
	mux.HandleFunc("POST /admin/orgs/{id}/rotate-key", adminOrgs.RotateAPIKey)

	// Provider key management
	if deps.ProviderKeyManager != nil {
		adminProviderKeys := NewAdminProviderKeysHandler(deps.ProviderKeyManager, deps.ProviderRegistry)
		mux.HandleFunc("GET /admin/orgs/{id}/provider-keys", adminProviderKeys.List)
		mux.HandleFunc("POST /admin/orgs/{id}/provider-keys", adminProviderKeys.Create)
		mux.HandleFunc("DELETE /admin/orgs/{id}/provider-keys/{keyId}", adminProviderKeys.Delete)
	}

	// Provider info
	if deps.ProviderRegistry != nil {
		mux.HandleFunc("GET /providers", handleListProviders(deps.ProviderRegistry))
	}
}

func methodNotAllowedHandler(allowedMethods string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Allow", allowedMethods)
		writeProxyError(w, http.StatusMethodNotAllowed, "method not allowed", "invalid_request_error")
	}
}
