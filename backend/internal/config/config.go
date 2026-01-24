package config

import (
	"fmt"
	"net/url"
	"os"
	"strings"
)

// ProviderConfig holds the configuration for the upstream AI provider.
// This is temporary MVP configuration using environment variables.
// Future: will be replaced by BYOK vault or per-org provider keys.
type ProviderConfig struct {
	BaseURL string
	APIKey  string
}

type Config struct {
	Port        string
	Environment string
	Provider    ProviderConfig
}

// Load reads configuration from environment variables.
// It fails fast with clear errors for missing required values.
func Load() (*Config, error) {
	var missing []string

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080" // sensible default
	}

	env := os.Getenv("ENV")
	if env == "" {
		env = "development" // sensible default
	}

	// Validate environment value
	if env != "development" && env != "staging" && env != "production" {
		return nil, fmt.Errorf("invalid ENV value %q: must be development, staging, or production", env)
	}

	// Load provider configuration (required for MVP)
	providerBaseURL := os.Getenv("PROVIDER_BASE_URL")
	if providerBaseURL == "" {
		missing = append(missing, "PROVIDER_BASE_URL")
	}

	providerAPIKey := os.Getenv("PROVIDER_API_KEY")
	if providerAPIKey == "" {
		missing = append(missing, "PROVIDER_API_KEY")
	}

	if len(missing) > 0 {
		return nil, fmt.Errorf("missing required environment variables: %v", missing)
	}

	// Validate provider base URL format
	if err := validateBaseURL(providerBaseURL); err != nil {
		return nil, fmt.Errorf("invalid PROVIDER_BASE_URL: %w", err)
	}

	// Validate provider API key format
	if err := validateAPIKey(providerAPIKey); err != nil {
		return nil, fmt.Errorf("invalid PROVIDER_API_KEY: %w", err)
	}

	return &Config{
		Port:        port,
		Environment: env,
		Provider: ProviderConfig{
			BaseURL: providerBaseURL,
			APIKey:  providerAPIKey,
		},
	}, nil
}

// validateBaseURL ensures the provider base URL is a valid HTTP/HTTPS URL.
func validateBaseURL(baseURL string) error {
	parsed, err := url.Parse(baseURL)
	if err != nil {
		return fmt.Errorf("malformed URL: %w", err)
	}

	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return fmt.Errorf("URL must use http or https scheme, got %q", parsed.Scheme)
	}

	if parsed.Host == "" {
		return fmt.Errorf("URL must include a host")
	}

	return nil
}

// validateAPIKey performs basic sanity checks on the API key format.
// This helps catch obvious configuration mistakes early.
func validateAPIKey(apiKey string) error {
	// Remove whitespace that might have been accidentally included
	trimmed := strings.TrimSpace(apiKey)
	
	if trimmed != apiKey {
		return fmt.Errorf("API key contains leading or trailing whitespace")
	}

	// Basic length check - most API keys are at least 20 characters
	if len(apiKey) < 20 {
		return fmt.Errorf("API key appears invalid (too short, must be at least 20 characters)")
	}

	// Check for common placeholder values
	lower := strings.ToLower(apiKey)
	placeholders := []string{"your-api-key", "sk-your-", "replace-me", "changeme", "example"}
	for _, placeholder := range placeholders {
		if strings.Contains(lower, placeholder) {
			return fmt.Errorf("API key appears to be a placeholder value")
		}
	}

	return nil
}
