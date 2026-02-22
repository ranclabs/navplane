package config

import (
	"encoding/base64"
	"fmt"
	"net/url"
	"os"
	"strconv"
	"strings"
)

// Auth0Config holds Auth0 JWT verification configuration.
type Auth0Config struct {
	Domain   string // e.g., "your-tenant.auth0.com"
	Audience string // e.g., "https://api.navplane.io"
}

// EncryptionConfig holds encryption key configuration for BYOK.
type EncryptionConfig struct {
	Key    []byte // 32-byte key for AES-256-GCM
	KeyNew []byte // Optional: new key for rotation
}

// DatabaseConfig holds PostgreSQL connection configuration.
type DatabaseConfig struct {
	URL             string
	MaxOpenConns    int
	MaxIdleConns    int
	ConnMaxLifetime int // seconds
}

type Config struct {
	Port        string
	Environment string
	Database    DatabaseConfig
	Auth0       Auth0Config
	Encryption  EncryptionConfig
}

// Load reads configuration from environment variables.
// It fails fast with clear errors for missing required values.
func Load() (*Config, error) {
	var missing []string

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	env := os.Getenv("ENV")
	if env == "" {
		env = "development"
	}

	if env != "development" && env != "staging" && env != "production" {
		return nil, fmt.Errorf("invalid ENV value %q: must be development, staging, or production", env)
	}

	// Database configuration (required)
	databaseURL := os.Getenv("DATABASE_URL")
	if databaseURL == "" {
		missing = append(missing, "DATABASE_URL")
	}

	// Auth0 configuration (required)
	auth0Domain := os.Getenv("AUTH0_DOMAIN")
	if auth0Domain == "" {
		missing = append(missing, "AUTH0_DOMAIN")
	}

	auth0Audience := os.Getenv("AUTH0_AUDIENCE")
	if auth0Audience == "" {
		missing = append(missing, "AUTH0_AUDIENCE")
	}

	// Encryption key (required)
	encryptionKeyB64 := os.Getenv("ENCRYPTION_KEY")
	if encryptionKeyB64 == "" {
		missing = append(missing, "ENCRYPTION_KEY")
	}

	if len(missing) > 0 {
		return nil, fmt.Errorf("missing required environment variables: %v", missing)
	}

	// Validate database URL format
	if err := validateDatabaseURL(databaseURL); err != nil {
		return nil, fmt.Errorf("invalid DATABASE_URL: %w", err)
	}

	// Validate and decode encryption key
	encryptionKey, err := decodeEncryptionKey(encryptionKeyB64)
	if err != nil {
		return nil, fmt.Errorf("invalid ENCRYPTION_KEY: %w", err)
	}

	// Optional: new encryption key for rotation
	var encryptionKeyNew []byte
	if keyNewB64 := os.Getenv("ENCRYPTION_KEY_NEW"); keyNewB64 != "" {
		encryptionKeyNew, err = decodeEncryptionKey(keyNewB64)
		if err != nil {
			return nil, fmt.Errorf("invalid ENCRYPTION_KEY_NEW: %w", err)
		}
	}

	// Validate Auth0 domain format
	if err := validateAuth0Domain(auth0Domain); err != nil {
		return nil, fmt.Errorf("invalid AUTH0_DOMAIN: %w", err)
	}

	dbConfig := DatabaseConfig{
		URL:             databaseURL,
		MaxOpenConns:    getEnvInt("DB_MAX_OPEN_CONNS", 25),
		MaxIdleConns:    getEnvInt("DB_MAX_IDLE_CONNS", 5),
		ConnMaxLifetime: getEnvInt("DB_CONN_MAX_LIFETIME", 300),
	}

	return &Config{
		Port:        port,
		Environment: env,
		Database:    dbConfig,
		Auth0: Auth0Config{
			Domain:   auth0Domain,
			Audience: auth0Audience,
		},
		Encryption: EncryptionConfig{
			Key:    encryptionKey,
			KeyNew: encryptionKeyNew,
		},
	}, nil
}

// decodeEncryptionKey decodes and validates a base64-encoded 32-byte key.
func decodeEncryptionKey(b64Key string) ([]byte, error) {
	key, err := base64.StdEncoding.DecodeString(strings.TrimSpace(b64Key))
	if err != nil {
		return nil, fmt.Errorf("must be valid base64: %w", err)
	}
	if len(key) != 32 {
		return nil, fmt.Errorf("must be exactly 32 bytes (256 bits), got %d bytes", len(key))
	}
	return key, nil
}

// validateAuth0Domain ensures the Auth0 domain is properly formatted.
func validateAuth0Domain(domain string) error {
	domain = strings.TrimSpace(domain)
	if domain == "" {
		return fmt.Errorf("domain cannot be empty")
	}

	// Should not include protocol
	if strings.HasPrefix(domain, "http://") || strings.HasPrefix(domain, "https://") {
		return fmt.Errorf("domain should not include protocol (http:// or https://)")
	}

	// Should look like a domain
	if !strings.Contains(domain, ".") {
		return fmt.Errorf("domain must be a valid hostname (e.g., your-tenant.auth0.com)")
	}

	return nil
}

// validateDatabaseURL ensures the database URL is a valid PostgreSQL connection string.
func validateDatabaseURL(dbURL string) error {
	parsed, err := url.Parse(dbURL)
	if err != nil {
		return fmt.Errorf("malformed URL: %w", err)
	}

	if parsed.Scheme != "postgres" && parsed.Scheme != "postgresql" {
		return fmt.Errorf("URL must use postgres or postgresql scheme, got %q", parsed.Scheme)
	}

	if parsed.Host == "" {
		return fmt.Errorf("URL must include a host")
	}

	return nil
}

// getEnvInt reads an environment variable as an integer with a default fallback.
func getEnvInt(key string, defaultVal int) int {
	val := os.Getenv(key)
	if val == "" {
		return defaultVal
	}
	intVal, err := strconv.Atoi(val)
	if err != nil {
		return defaultVal
	}
	return intVal
}
