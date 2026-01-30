// Package middleware provides HTTP middleware for NavPlane.
package middleware

import (
	"context"
	"errors"
	"net/http"

	"navplane/internal/auth"
)

// contextKey is a custom type for context keys to avoid collisions.
type contextKey string

// OrgContextKey is the context key for storing org information.
const OrgContextKey contextKey = "org"

// GetOrgContext retrieves the authenticated org context from the request context.
// Returns the OrgContext and true if found, nil and false otherwise.
func GetOrgContext(ctx context.Context) (*auth.OrgContext, bool) {
	org, ok := ctx.Value(OrgContextKey).(*auth.OrgContext)
	return org, ok
}

// RequireAuth returns middleware that authenticates requests using the provided AuthStore.
// It extracts the bearer token, validates it, and attaches the org context to the request.
//
// Authentication flow:
//  1. Extract bearer token from Authorization header
//  2. Validate token against the auth store
//  3. Check if org is enabled (kill switch)
//  4. Attach org context to request and continue
//
// Error responses:
//   - 401 Unauthorized: Missing or malformed Authorization header
//   - 403 Forbidden: Invalid token or disabled org
//   - 500 Internal Server Error: Database or other server error
func RequireAuth(store auth.AuthStore) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// 1. Extract bearer token
			token, err := auth.ExtractBearerToken(r)
			if err != nil {
				// Missing or malformed Authorization header
				auth.WriteUnauthorized(w)
				return
			}

			// 2. Validate token against the store
			orgCtx, err := store.ValidateToken(r.Context(), token)
			if err != nil {
				if errors.Is(err, auth.ErrTokenNotFound) {
					// Invalid token
					auth.WriteForbidden(w)
					return
				}
				// Database or other server error - don't leak details
				auth.WriteJSONError(w, http.StatusInternalServerError, "internal error", "server_error")
				return
			}

			// 3. Check if org is enabled (kill switch)
			if !orgCtx.Enabled {
				// Org is disabled - treat same as forbidden
				auth.WriteForbidden(w)
				return
			}

			// 4. Attach org context to request and continue
			ctx := context.WithValue(r.Context(), OrgContextKey, orgCtx)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}
