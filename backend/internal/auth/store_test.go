package auth

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"testing"
)

func TestHashToken(t *testing.T) {
	// Test that HashToken produces consistent SHA-256 hashes
	token := "np-test-token-12345"
	
	// Compute expected hash manually
	hash := sha256.Sum256([]byte(token))
	expected := hex.EncodeToString(hash[:])
	
	result := HashToken(token)
	
	if result != expected {
		t.Errorf("HashToken(%q) = %q, expected %q", token, result, expected)
	}
}

func TestHashToken_DifferentTokensDifferentHashes(t *testing.T) {
	token1 := "np-token-aaa"
	token2 := "np-token-bbb"
	
	hash1 := HashToken(token1)
	hash2 := HashToken(token2)
	
	if hash1 == hash2 {
		t.Errorf("different tokens should produce different hashes")
	}
}

func TestHashToken_SameTokenSameHash(t *testing.T) {
	token := "np-consistent-token"
	
	hash1 := HashToken(token)
	hash2 := HashToken(token)
	
	if hash1 != hash2 {
		t.Errorf("same token should produce same hash: %q != %q", hash1, hash2)
	}
}

func TestHashToken_EmptyToken(t *testing.T) {
	// Empty token should still produce a valid hash
	result := HashToken("")
	
	// SHA-256 of empty string is a known value
	expected := "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855"
	
	if result != expected {
		t.Errorf("HashToken(\"\") = %q, expected %q", result, expected)
	}
}

func TestHashToken_Length(t *testing.T) {
	// SHA-256 produces a 64-character hex string (256 bits = 32 bytes = 64 hex chars)
	result := HashToken("any-token")
	
	if len(result) != 64 {
		t.Errorf("HashToken result length = %d, expected 64", len(result))
	}
}

// MockAuthStore implements AuthStore for testing purposes.
type MockAuthStore struct {
	ValidateFunc func(ctx context.Context, token string) (*OrgContext, error)
	CloseFunc    func() error
}

func (m *MockAuthStore) ValidateToken(ctx context.Context, token string) (*OrgContext, error) {
	if m.ValidateFunc != nil {
		return m.ValidateFunc(ctx, token)
	}
	return nil, ErrTokenNotFound
}

func (m *MockAuthStore) Close() error {
	if m.CloseFunc != nil {
		return m.CloseFunc()
	}
	return nil
}

func TestMockAuthStore_ValidToken(t *testing.T) {
	// Test that mock store can simulate valid token
	store := &MockAuthStore{
		ValidateFunc: func(ctx context.Context, token string) (*OrgContext, error) {
			if token == "valid-token" {
				return &OrgContext{
					OrgID:   "org-123",
					OrgName: "Test Org",
					Enabled: true,
				}, nil
			}
			return nil, ErrTokenNotFound
		},
	}
	
	ctx := context.Background()
	
	// Valid token
	orgCtx, err := store.ValidateToken(ctx, "valid-token")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if orgCtx.OrgID != "org-123" {
		t.Errorf("OrgID = %q, expected %q", orgCtx.OrgID, "org-123")
	}
	if orgCtx.OrgName != "Test Org" {
		t.Errorf("OrgName = %q, expected %q", orgCtx.OrgName, "Test Org")
	}
	if !orgCtx.Enabled {
		t.Error("Enabled should be true")
	}
	
	// Invalid token
	_, err = store.ValidateToken(ctx, "invalid-token")
	if err != ErrTokenNotFound {
		t.Errorf("expected ErrTokenNotFound, got %v", err)
	}
}

func TestErrTokenNotFound(t *testing.T) {
	// Verify error message is generic (doesn't leak details)
	errMsg := ErrTokenNotFound.Error()
	
	if errMsg != "token not found" {
		t.Errorf("error message = %q, expected 'token not found'", errMsg)
	}
}
