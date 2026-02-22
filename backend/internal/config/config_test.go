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
	t.Setenv("PROVIDER_BASE_URL", "") // Explicitly unset
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
	t.Setenv("PROVIDER_API_KEY", "") // Explicitly unset
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
	t.Setenv("PROVIDER_BASE_URL", "") // Explicitly unset
	t.Setenv("PROVIDER_API_KEY", "")  // Explicitly unset
	t.Setenv("DATABASE_URL", "postgres://user:pass@localhost:5432/testdb?sslmode=disable")

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

func TestLoad_MissingDatabaseURL(t *testing.T) {
	t.Setenv("PROVIDER_BASE_URL", "https://api.openai.com/v1")
	t.Setenv("PROVIDER_API_KEY", "sk-test-key-12345678901234567890")
	t.Setenv("DATABASE_URL", "") // Explicitly unset

	_, err := Load()
	if err == nil {
		t.Fatal("expected error for missing DATABASE_URL, got nil")
	}

	if !strings.Contains(err.Error(), "DATABASE_URL") {
		t.Errorf("error message should mention DATABASE_URL, got: %v", err)
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

	// Database pool defaults
	if cfg.Database.MaxOpenConns != 25 {
		t.Errorf("expected default MaxOpenConns to be 25, got: %d", cfg.Database.MaxOpenConns)
	}
	if cfg.Database.MaxIdleConns != 5 {
		t.Errorf("expected default MaxIdleConns to be 5, got: %d", cfg.Database.MaxIdleConns)
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

func TestLoad_InvalidDatabaseURL(t *testing.T) {
	tests := []struct {
		name        string
		dbURL       string
		expectError string
	}{
		{
			name:        "missing scheme",
			dbURL:       "localhost:5432/testdb",
			expectError: "URL must use postgres or postgresql scheme",
		},
		{
			name:        "invalid scheme",
			dbURL:       "mysql://user:pass@localhost:3306/testdb",
			expectError: "URL must use postgres or postgresql scheme",
		},
		{
			name:        "missing host",
			dbURL:       "postgres://",
			expectError: "URL must include a host",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv("PROVIDER_BASE_URL", "https://api.openai.com/v1")
			t.Setenv("PROVIDER_API_KEY", "sk-test-key-12345678901234567890")
			t.Setenv("DATABASE_URL", tt.dbURL)

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
