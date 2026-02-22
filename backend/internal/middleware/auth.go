package middleware

import (
	"context"
	"errors"
	"log"
	"net/http"
	"strings"

	"lectr/internal/org"
)

// contextKey is a custom type for context keys to avoid collisions.
type contextKey string

const (
	// OrgContextKey is the context key for the authenticated organization.
	OrgContextKey contextKey = "org"
)

// Auth creates authentication middleware that validates Lectr API keys.
// Extracts Bearer token from Authorization header, authenticates via org manager,
// and injects the org into the request context.
func Auth(manager *org.Manager) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			token, err := extractBearerToken(r)
			if err != nil {
				writeAuthError(w, http.StatusUnauthorized, "missing or invalid authorization header")
				return
			}

			authenticatedOrg, err := manager.Authenticate(r.Context(), token)
			if err != nil {
				if errors.Is(err, org.ErrNotFound) || errors.Is(err, org.ErrInvalidKey) {
					writeAuthError(w, http.StatusUnauthorized, "invalid API key")
					return
				}
				if errors.Is(err, org.ErrOrgDisabled) {
					writeAuthError(w, http.StatusForbidden, "organization is disabled")
					return
				}
				log.Printf("authentication error: %v", err)
				writeAuthError(w, http.StatusInternalServerError, "authentication failed")
				return
			}

			ctx := context.WithValue(r.Context(), OrgContextKey, authenticatedOrg)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// GetOrg retrieves the authenticated organization from the request context.
// Returns nil if no org is in context (middleware not applied or auth failed).
func GetOrg(ctx context.Context) *org.Org {
	o, ok := ctx.Value(OrgContextKey).(*org.Org)
	if !ok {
		return nil
	}
	return o
}

// extractBearerToken extracts the token from "Authorization: Bearer <token>" header.
func extractBearerToken(r *http.Request) (string, error) {
	authHeader := r.Header.Get("Authorization")
	if authHeader == "" {
		return "", errors.New("missing authorization header")
	}

	const prefix = "Bearer "
	if !strings.HasPrefix(authHeader, prefix) {
		return "", errors.New("invalid authorization scheme")
	}

	token := strings.TrimPrefix(authHeader, prefix)
	if token == "" {
		return "", errors.New("empty bearer token")
	}

	return token, nil
}

// writeAuthError writes an OpenAI-compatible authentication error response.
func writeAuthError(w http.ResponseWriter, status int, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	// Using simple string concatenation to avoid import cycle with auth package
	response := `{"error":{"message":"` + message + `","type":"authentication_error"}}`
	if _, err := w.Write([]byte(response)); err != nil {
		log.Printf("failed to write auth error response: %v", err)
	}
}
