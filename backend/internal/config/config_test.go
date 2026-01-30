package config

import (
	"strings"
	"testing"
)

func TestLoad_Success(t *testing.T) {
	// Set required environment variables
	t.Setenv("PROVIDER_BASE_URL", "https://api.openai.com/v1")
	t.Setenv("PROVIDER_API_KEY", "sk-test-key-12345678901234567890")
	t.Setenv("DATABASE_URL", "postgres://user:pass@localhost:5432/testdb?sslmode=disable")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	if cfg.Provider.BaseURL != "https://api.openai.com/v1" {
		t.Errorf("expected BaseURL to be 'https://api.openai.com/v1', got: %s", cfg.Provider.BaseURL)
	}

	if cfg.Provider.APIKey != "sk-test-key-12345678901234567890" {
		t.Errorf("expected APIKey to be 'sk-test-key-12345678901234567890', got: %s", cfg.Provider.APIKey)
	}

	if cfg.Database.URL != "postgres://user:pass@localhost:5432/testdb?sslmode=disable" {
		t.Errorf("expected Database.URL to be set, got: %s", cfg.Database.URL)
	}
}

func TestLoad_MissingProviderBaseURL(t *testing.T) {
	t.Setenv("PROVIDER_API_KEY", "sk-test-key-12345678901234567890")
	t.Setenv("DATABASE_URL", "postgres://user:pass@localhost:5432/testdb?sslmode=disable")

	_, err := Load()
	if err == nil {
		t.Fatal("expected error for missing PROVIDER_BASE_URL, got nil")
	}

	if !strings.Contains(err.Error(), "PROVIDER_BASE_URL") {
		t.Errorf("error message should mention PROVIDER_BASE_URL, got: %v", err)
	}
}

func TestLoad_MissingProviderAPIKey(t *testing.T) {
	t.Setenv("PROVIDER_BASE_URL", "https://api.openai.com/v1")
	t.Setenv("DATABASE_URL", "postgres://user:pass@localhost:5432/testdb?sslmode=disable")

	_, err := Load()
	if err == nil {
		t.Fatal("expected error for missing PROVIDER_API_KEY, got nil")
	}

	if !strings.Contains(err.Error(), "PROVIDER_API_KEY") {
		t.Errorf("error message should mention PROVIDER_API_KEY, got: %v", err)
	}
}

func TestLoad_MissingBothProviderVars(t *testing.T) {
	// Don't set any provider env vars, allowing the test to check for missing vars

	_, err := Load()
	if err == nil {
		t.Fatal("expected error for missing provider environment variables, got nil")
	}

	// Both should be mentioned in the error
	if !strings.Contains(err.Error(), "PROVIDER_BASE_URL") {
		t.Errorf("error message should mention PROVIDER_BASE_URL, got: %v", err)
	}
	if !strings.Contains(err.Error(), "PROVIDER_API_KEY") {
		t.Errorf("error message should mention PROVIDER_API_KEY, got: %v", err)
	}
}

func TestLoad_WithDefaults(t *testing.T) {
	t.Setenv("PROVIDER_BASE_URL", "https://api.openai.com/v1")
	t.Setenv("PROVIDER_API_KEY", "sk-test-key-12345678901234567890")
	t.Setenv("DATABASE_URL", "postgres://user:pass@localhost:5432/testdb?sslmode=disable")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	// PORT should default to 8080
	if cfg.Port != "8080" {
		t.Errorf("expected default Port to be '8080', got: %s", cfg.Port)
	}

	// ENV should default to development
	if cfg.Environment != "development" {
		t.Errorf("expected default Environment to be 'development', got: %s", cfg.Environment)
	}
}

func TestLoad_InvalidBaseURL(t *testing.T) {
	tests := []struct {
		name        string
		baseURL     string
		expectError string
	}{
		{
			name:        "malformed URL",
			baseURL:     "ht!tp://invalid url",
			expectError: "malformed URL",
		},
		{
			name:        "missing scheme",
			baseURL:     "api.openai.com/v1",
			expectError: "URL must use http or https scheme",
		},
		{
			name:        "invalid scheme",
			baseURL:     "ftp://api.openai.com/v1",
			expectError: "URL must use http or https scheme",
		},
		{
			name:        "missing host",
			baseURL:     "https://",
			expectError: "URL must include a host",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv("PROVIDER_BASE_URL", tt.baseURL)
			t.Setenv("PROVIDER_API_KEY", "sk-test-key-12345678901234567890")
			t.Setenv("DATABASE_URL", "postgres://user:pass@localhost:5432/testdb?sslmode=disable")

			_, err := Load()
			if err == nil {
				t.Fatal("expected error, got nil")
			}

			if !strings.Contains(err.Error(), tt.expectError) {
				t.Errorf("expected error containing %q, got: %v", tt.expectError, err)
			}
		})
	}
}

func TestLoad_InvalidAPIKey(t *testing.T) {
	tests := []struct {
		name        string
		apiKey      string
		expectError string
	}{
		{
			name:        "too short",
			apiKey:      "sk-short",
			expectError: "too short",
		},
		{
			name:        "leading whitespace",
			apiKey:      " sk-test-key-12345678901234567890",
			expectError: "whitespace",
		},
		{
			name:        "trailing whitespace",
			apiKey:      "sk-test-key-12345678901234567890 ",
			expectError: "whitespace",
		},
		{
			name:        "placeholder value 1",
			apiKey:      "your-api-key-here-replace-this",
			expectError: "placeholder",
		},
		{
			name:        "placeholder value 2",
			apiKey:      "sk-your-actual-api-key-here",
			expectError: "placeholder",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv("PROVIDER_BASE_URL", "https://api.openai.com/v1")
			t.Setenv("PROVIDER_API_KEY", tt.apiKey)
			t.Setenv("DATABASE_URL", "postgres://user:pass@localhost:5432/testdb?sslmode=disable")

			_, err := Load()
			if err == nil {
				t.Fatal("expected error, got nil")
			}

			if !strings.Contains(err.Error(), tt.expectError) {
				t.Errorf("expected error containing %q, got: %v", tt.expectError, err)
			}
		})
	}
}

func TestLoad_MissingDatabaseURL(t *testing.T) {
	t.Setenv("PROVIDER_BASE_URL", "https://api.openai.com/v1")
	t.Setenv("PROVIDER_API_KEY", "sk-test-key-12345678901234567890")

	_, err := Load()
	if err == nil {
		t.Fatal("expected error for missing DATABASE_URL, got nil")
	}

	if !strings.Contains(err.Error(), "DATABASE_URL") {
		t.Errorf("error message should mention DATABASE_URL, got: %v", err)
	}
}

func TestLoad_InvalidDatabaseURL(t *testing.T) {
	tests := []struct {
		name        string
		databaseURL string
	}{
		{
			name:        "mysql scheme",
			databaseURL: "mysql://user:pass@localhost:3306/db",
		},
		{
			name:        "http scheme",
			databaseURL: "http://localhost:5432/db",
		},
		{
			name:        "no scheme",
			databaseURL: "localhost:5432/db",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv("PROVIDER_BASE_URL", "https://api.openai.com/v1")
			t.Setenv("PROVIDER_API_KEY", "sk-test-key-12345678901234567890")
			t.Setenv("DATABASE_URL", tt.databaseURL)

			_, err := Load()
			if err == nil {
				t.Fatal("expected error for invalid DATABASE_URL, got nil")
			}

			if !strings.Contains(err.Error(), "postgres://") {
				t.Errorf("error message should mention postgres://, got: %v", err)
			}
		})
	}
}

func TestLoad_ValidDatabaseURL(t *testing.T) {
	tests := []struct {
		name        string
		databaseURL string
	}{
		{
			name:        "postgres scheme",
			databaseURL: "postgres://user:pass@localhost:5432/db?sslmode=disable",
		},
		{
			name:        "postgresql scheme",
			databaseURL: "postgresql://user:pass@localhost:5432/db?sslmode=disable",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv("PROVIDER_BASE_URL", "https://api.openai.com/v1")
			t.Setenv("PROVIDER_API_KEY", "sk-test-key-12345678901234567890")
			t.Setenv("DATABASE_URL", tt.databaseURL)

			cfg, err := Load()
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if cfg.Database.URL != tt.databaseURL {
				t.Errorf("expected Database.URL = %q, got %q", tt.databaseURL, cfg.Database.URL)
			}
		})
	}
}
