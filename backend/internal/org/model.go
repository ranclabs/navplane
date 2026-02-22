package org

import (
	"crypto/sha256"
	"encoding/hex"
	"time"

	"github.com/google/uuid"
)

// Org represents an organization in NavPlane.
// Organizations are the top-level tenant for API access.
type Org struct {
	ID         uuid.UUID
	Name       string
	APIKeyHash string
	Enabled    bool
	CreatedAt  time.Time
	UpdatedAt  time.Time
}

// APIKey represents a generated API key before hashing.
// The plaintext key is only available at creation time.
type APIKey struct {
	Prefix    string // First 8 chars for identification (e.g., "np_live_")
	Plaintext string // Full key - only shown once at creation
	Hash      string // SHA-256 hash stored in database
}

// GenerateAPIKey creates a new API key with a NavPlane prefix.
// Returns the full key (to show user once) and the hash (to store).
func GenerateAPIKey() APIKey {
	id := uuid.New()
	plaintext := "np_" + id.String()
	hash := HashAPIKey(plaintext)

	return APIKey{
		Prefix:    plaintext[:11], // "np_" + first 8 chars of UUID
		Plaintext: plaintext,
		Hash:      hash,
	}
}

// HashAPIKey creates a SHA-256 hash of an API key for secure storage.
func HashAPIKey(key string) string {
	h := sha256.Sum256([]byte(key))
	return hex.EncodeToString(h[:])
}
