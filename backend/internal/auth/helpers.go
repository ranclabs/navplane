// Package auth provides authentication helpers for NavPlane.
package auth

import (
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"strings"
)

// Sentinel errors for token extraction failures.
// These can be used for debugging/logging but should NOT be exposed in responses.
var (
	ErrMissingAuthHeader = errors.New("missing authorization header")
	ErrInvalidAuthScheme = errors.New("invalid authorization scheme: expected Bearer")
	ErrEmptyToken        = errors.New("empty bearer token")
)

// ExtractBearerToken extracts the token from an "Authorization: Bearer <token>" header.
// Returns the token string on success.
// Returns an error if the header is missing, uses wrong scheme, or token is empty.
// Does not log anything.
func ExtractBearerToken(r *http.Request) (string, error) {
	authHeader := r.Header.Get("Authorization")
	if authHeader == "" {
		return "", ErrMissingAuthHeader
	}

	// Must start with "Bearer "
	const prefix = "Bearer "
	if !strings.HasPrefix(authHeader, prefix) {
		return "", ErrInvalidAuthScheme
	}

	token := strings.TrimPrefix(authHeader, prefix)
	if token == "" {
		return "", ErrEmptyToken
	}

	return token, nil
}

// APIError represents an OpenAI-compatible error response.
type APIError struct {
	Error ErrorDetail `json:"error"`
}

// ErrorDetail contains the error message and type.
type ErrorDetail struct {
	Message string `json:"message"`
	Type    string `json:"type"`
}

// WriteJSONError writes an OpenAI-compatible JSON error response.
// Always sets Content-Type: application/json.
// Response format: {"error": {"message": "<message>", "type": "<errorType>"}}
func WriteJSONError(w http.ResponseWriter, status int, message, errorType string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(APIError{
		Error: ErrorDetail{
			Message: message,
			Type:    errorType,
		},
	}); err != nil {
		log.Printf("failed to write JSON error response: %v", err)
	}
}

// WriteUnauthorized writes a 401 Unauthorized JSON response.
// Use when Authorization header is missing or malformed.
func WriteUnauthorized(w http.ResponseWriter) {
	WriteJSONError(w, http.StatusUnauthorized, "unauthorized", "authentication_error")
}

// WriteForbidden writes a 403 Forbidden JSON response.
// Use when token is invalid (not found, expired, disabled org, etc.).
func WriteForbidden(w http.ResponseWriter) {
	WriteJSONError(w, http.StatusForbidden, "forbidden", "authentication_error")
}
