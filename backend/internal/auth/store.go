package auth

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"errors"
	"fmt"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib" // PostgreSQL driver
)

// ErrTokenNotFound is returned when the token does not exist or is invalid.
var ErrTokenNotFound = errors.New("token not found")

// OrgContext contains information about the authenticated organization.
// This is attached to the request context after successful authentication.
type OrgContext struct {
	OrgID   string
	OrgName string
	Enabled bool
}

// AuthStore defines the interface for token validation.
// This abstraction allows for different implementations (e.g., mock for testing).
type AuthStore interface {
	// ValidateToken checks if a token is valid and returns the associated org info.
	// Returns ErrTokenNotFound if the token is invalid or does not exist.
	ValidateToken(ctx context.Context, token string) (*OrgContext, error)

	// Close releases any resources held by the store.
	Close() error
}

// PostgresAuthStore implements AuthStore using PostgreSQL.
type PostgresAuthStore struct {
	db *sql.DB
}

// NewPostgresAuthStore creates a new PostgresAuthStore with the given database connection.
func NewPostgresAuthStore(db *sql.DB) *PostgresAuthStore {
	return &PostgresAuthStore{db: db}
}

// NewPostgresAuthStoreFromURL creates a new PostgresAuthStore by connecting to the given URL.
func NewPostgresAuthStoreFromURL(databaseURL string) (*PostgresAuthStore, error) {
	db, err := sql.Open("pgx", databaseURL)
	if err != nil {
		return nil, err
	}

	// Verify connection with timeout to prevent hanging if DB is unreachable
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := db.PingContext(ctx); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}

	return &PostgresAuthStore{db: db}, nil
}

// ValidateToken hashes the provided token and looks it up in the database.
// If found, returns the associated org context. Otherwise, returns ErrTokenNotFound.
//
// Security notes:
// - Token is hashed with SHA-256 before any database operation
// - Raw token is never logged or stored
// - Query uses indexed lookup on token_hash for fast authentication
func (s *PostgresAuthStore) ValidateToken(ctx context.Context, token string) (*OrgContext, error) {
	// Hash the token
	tokenHash := hashToken(token)

	// Query for the token and its associated org
	// Uses JOIN to get org details in a single query
	const query = `
		SELECT o.id, o.name, o.enabled
		FROM orgs o
		INNER JOIN org_api_tokens t ON t.org_id = o.id
		WHERE t.token_hash = $1
	`

	var orgCtx OrgContext
	err := s.db.QueryRowContext(ctx, query, tokenHash).Scan(
		&orgCtx.OrgID,
		&orgCtx.OrgName,
		&orgCtx.Enabled,
	)

	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrTokenNotFound
		}
		// Don't wrap the error with token details
		return nil, err
	}

	return &orgCtx, nil
}

// Close closes the database connection.
func (s *PostgresAuthStore) Close() error {
	return s.db.Close()
}

// hashToken computes SHA-256 hash of the token and returns it as a hex string.
// This is the same hashing method used when storing tokens.
func hashToken(token string) string {
	hash := sha256.Sum256([]byte(token))
	return hex.EncodeToString(hash[:])
}

// HashToken is exported for use when creating new tokens.
// Returns the SHA-256 hash of the token as a hex string.
func HashToken(token string) string {
	return hashToken(token)
}
