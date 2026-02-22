package org

import (
	"strings"
	"testing"
)

func TestGenerateAPIKey(t *testing.T) {
	key := GenerateAPIKey()

	// Check prefix
	if !strings.HasPrefix(key.Plaintext, "np_") {
		t.Errorf("expected key to start with 'np_', got %s", key.Plaintext)
	}

	// Check length (np_ + UUID = 3 + 36 = 39 chars)
	if len(key.Plaintext) != 39 {
		t.Errorf("expected key length 39, got %d", len(key.Plaintext))
	}

	// Check prefix field
	if len(key.Prefix) != 11 {
		t.Errorf("expected prefix length 11, got %d", len(key.Prefix))
	}
	if !strings.HasPrefix(key.Plaintext, key.Prefix) {
		t.Error("plaintext should start with prefix")
	}

	// Check hash is set and different from plaintext
	if key.Hash == "" {
		t.Error("hash should not be empty")
	}
	if key.Hash == key.Plaintext {
		t.Error("hash should be different from plaintext")
	}

	// Hash should be 64 chars (SHA-256 hex)
	if len(key.Hash) != 64 {
		t.Errorf("expected hash length 64 (SHA-256 hex), got %d", len(key.Hash))
	}
}

func TestGenerateAPIKey_Uniqueness(t *testing.T) {
	key1 := GenerateAPIKey()
	key2 := GenerateAPIKey()

	if key1.Plaintext == key2.Plaintext {
		t.Error("generated keys should be unique")
	}
	if key1.Hash == key2.Hash {
		t.Error("generated key hashes should be unique")
	}
}

func TestHashAPIKey(t *testing.T) {
	key := "np_test-key-12345"
	hash1 := HashAPIKey(key)
	hash2 := HashAPIKey(key)

	// Same input should produce same hash
	if hash1 != hash2 {
		t.Error("same key should produce same hash")
	}

	// Different input should produce different hash
	hash3 := HashAPIKey("np_different-key")
	if hash1 == hash3 {
		t.Error("different keys should produce different hashes")
	}

	// Hash should be 64 chars
	if len(hash1) != 64 {
		t.Errorf("expected hash length 64, got %d", len(hash1))
	}
}
