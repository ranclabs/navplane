package provider

import (
	"context"
	"errors"
)

// Provider represents an AI provider that NavPlane can route requests to.
type Provider interface {
	// Name returns the provider identifier (e.g., "openai", "anthropic")
	Name() string

	// DisplayName returns the human-readable name (e.g., "OpenAI", "Anthropic")
	DisplayName() string

	// BaseURL returns the default API base URL for this provider
	BaseURL() string

	// Models returns the list of supported models
	Models() []Model

	// ValidateKey tests if an API key is valid by making a lightweight API call
	ValidateKey(ctx context.Context, apiKey string) error

	// AuthHeader returns the header name used for authentication
	AuthHeader() string

	// FormatAuthValue formats the API key for the auth header
	FormatAuthValue(apiKey string) string
}

// Model represents a model supported by a provider.
type Model struct {
	ID           string   `json:"id"`
	Name         string   `json:"name"`
	Provider     string   `json:"provider"`
	ContextSize  int      `json:"context_size"`
	Capabilities []string `json:"capabilities"`
}

// Common errors
var (
	ErrProviderNotFound = errors.New("provider not found")
	ErrInvalidAPIKey    = errors.New("invalid API key")
	ErrKeyValidation    = errors.New("failed to validate API key")
)
