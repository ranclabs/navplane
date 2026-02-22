package config

import (
	"encoding/base64"
	"strings"
	"testing"
)

// validEncryptionKey is a base64-encoded 32-byte key for testing
var validEncryptionKey = base64.StdEncoding.EncodeToString(make([]byte, 32))

func setRequiredEnvVars(t *testing.T) {
	t.Setenv("DATABASE_URL", "postgres://user:pass@localhost:5432/testdb?sslmode=disable")
	t.Setenv("AUTH0_DOMAIN", "test-tenant.auth0.com")
	t.Setenv("AUTH0_AUDIENCE", "https://api.navplane.io")
	t.Setenv("ENCRYPTION_KEY", validEncryptionKey)
}

func TestLoad_Success(t *testing.T) {
	setRequiredEnvVars(t)

	cfg, err := Load()
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	if cfg.Database.URL != "postgres://user:pass@localhost:5432/testdb?sslmode=disable" {
		t.Errorf("expected Database.URL to be set, got: %s", cfg.Database.URL)
	}

	if cfg.Auth0.Domain != "test-tenant.auth0.com" {
		t.Errorf("expected Auth0.Domain to be 'test-tenant.auth0.com', got: %s", cfg.Auth0.Domain)
	}

	if cfg.Auth0.Audience != "https://api.navplane.io" {
		t.Errorf("expected Auth0.Audience to be 'https://api.navplane.io', got: %s", cfg.Auth0.Audience)
	}

	if len(cfg.Encryption.Key) != 32 {
		t.Errorf("expected Encryption.Key to be 32 bytes, got: %d", len(cfg.Encryption.Key))
	}
}

func TestLoad_MissingDatabaseURL(t *testing.T) {
	setRequiredEnvVars(t)
	t.Setenv("DATABASE_URL", "")

	_, err := Load()
	if err == nil {
		t.Fatal("expected error for missing DATABASE_URL, got nil")
	}

	if !strings.Contains(err.Error(), "DATABASE_URL") {
		t.Errorf("error message should mention DATABASE_URL, got: %v", err)
	}
}

func TestLoad_MissingAuth0Domain(t *testing.T) {
	setRequiredEnvVars(t)
	t.Setenv("AUTH0_DOMAIN", "")

	_, err := Load()
	if err == nil {
		t.Fatal("expected error for missing AUTH0_DOMAIN, got nil")
	}

	if !strings.Contains(err.Error(), "AUTH0_DOMAIN") {
		t.Errorf("error message should mention AUTH0_DOMAIN, got: %v", err)
	}
}

func TestLoad_MissingAuth0Audience(t *testing.T) {
	setRequiredEnvVars(t)
	t.Setenv("AUTH0_AUDIENCE", "")

	_, err := Load()
	if err == nil {
		t.Fatal("expected error for missing AUTH0_AUDIENCE, got nil")
	}

	if !strings.Contains(err.Error(), "AUTH0_AUDIENCE") {
		t.Errorf("error message should mention AUTH0_AUDIENCE, got: %v", err)
	}
}

func TestLoad_MissingEncryptionKey(t *testing.T) {
	setRequiredEnvVars(t)
	t.Setenv("ENCRYPTION_KEY", "")

	_, err := Load()
	if err == nil {
		t.Fatal("expected error for missing ENCRYPTION_KEY, got nil")
	}

	if !strings.Contains(err.Error(), "ENCRYPTION_KEY") {
		t.Errorf("error message should mention ENCRYPTION_KEY, got: %v", err)
	}
}

func TestLoad_WithDefaults(t *testing.T) {
	setRequiredEnvVars(t)

	cfg, err := Load()
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	if cfg.Port != "8080" {
		t.Errorf("expected default Port to be '8080', got: %s", cfg.Port)
	}

	if cfg.Environment != "development" {
		t.Errorf("expected default Environment to be 'development', got: %s", cfg.Environment)
	}

	if cfg.Database.MaxOpenConns != 25 {
		t.Errorf("expected default MaxOpenConns to be 25, got: %d", cfg.Database.MaxOpenConns)
	}
	if cfg.Database.MaxIdleConns != 5 {
		t.Errorf("expected default MaxIdleConns to be 5, got: %d", cfg.Database.MaxIdleConns)
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
			setRequiredEnvVars(t)
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

func TestLoad_InvalidEncryptionKey(t *testing.T) {
	tests := []struct {
		name        string
		key         string
		expectError string
	}{
		{
			name:        "not base64",
			key:         "not-valid-base64!!!",
			expectError: "must be valid base64",
		},
		{
			name:        "too short",
			key:         base64.StdEncoding.EncodeToString(make([]byte, 16)),
			expectError: "must be exactly 32 bytes",
		},
		{
			name:        "too long",
			key:         base64.StdEncoding.EncodeToString(make([]byte, 64)),
			expectError: "must be exactly 32 bytes",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			setRequiredEnvVars(t)
			t.Setenv("ENCRYPTION_KEY", tt.key)

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

func TestLoad_InvalidAuth0Domain(t *testing.T) {
	tests := []struct {
		name        string
		domain      string
		expectError string
	}{
		{
			name:        "includes http",
			domain:      "http://test.auth0.com",
			expectError: "should not include protocol",
		},
		{
			name:        "includes https",
			domain:      "https://test.auth0.com",
			expectError: "should not include protocol",
		},
		{
			name:        "not a valid hostname",
			domain:      "localhost",
			expectError: "must be a valid hostname",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			setRequiredEnvVars(t)
			t.Setenv("AUTH0_DOMAIN", tt.domain)

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

func TestLoad_WithEncryptionKeyNew(t *testing.T) {
	setRequiredEnvVars(t)
	newKey := base64.StdEncoding.EncodeToString(make([]byte, 32))
	t.Setenv("ENCRYPTION_KEY_NEW", newKey)

	cfg, err := Load()
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	if len(cfg.Encryption.KeyNew) != 32 {
		t.Errorf("expected Encryption.KeyNew to be 32 bytes, got: %d", len(cfg.Encryption.KeyNew))
	}
}

func TestLoad_InvalidEnvironment(t *testing.T) {
	setRequiredEnvVars(t)
	t.Setenv("ENV", "invalid")

	_, err := Load()
	if err == nil {
		t.Fatal("expected error for invalid ENV, got nil")
	}

	if !strings.Contains(err.Error(), "invalid ENV value") {
		t.Errorf("error message should mention invalid ENV, got: %v", err)
	}
}
