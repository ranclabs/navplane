package jwtauth

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

func TestNewVerifier(t *testing.T) {
	tests := []struct {
		name    string
		cfg     Config
		wantErr bool
	}{
		{
			name:    "valid config",
			cfg:     Config{Domain: "test.auth0.com", Audience: "https://api.test.com"},
			wantErr: false,
		},
		{
			name:    "missing domain",
			cfg:     Config{Audience: "https://api.test.com"},
			wantErr: true,
		},
		{
			name:    "missing audience",
			cfg:     Config{Domain: "test.auth0.com"},
			wantErr: true,
		},
		{
			name:    "domain with https prefix",
			cfg:     Config{Domain: "https://test.auth0.com", Audience: "https://api.test.com"},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := NewVerifier(tt.cfg)
			if (err != nil) != tt.wantErr {
				t.Errorf("NewVerifier() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestClaims_Auth0UserID(t *testing.T) {
	claims := &Claims{}
	claims.Subject = "auth0|123456"

	if got := claims.Auth0UserID(); got != "auth0|123456" {
		t.Errorf("Auth0UserID() = %v, want %v", got, "auth0|123456")
	}
}

func TestClaims_HasPermission(t *testing.T) {
	claims := &Claims{
		Permissions: []string{"read:users", "write:users"},
	}

	if !claims.HasPermission("read:users") {
		t.Error("expected HasPermission(read:users) to be true")
	}

	if claims.HasPermission("delete:users") {
		t.Error("expected HasPermission(delete:users) to be false")
	}
}

func TestGetClaims(t *testing.T) {
	claims := &Claims{}
	claims.Subject = "test-user"

	ctx := context.WithValue(context.Background(), ClaimsContextKey, claims)
	got := GetClaims(ctx)

	if got == nil {
		t.Fatal("GetClaims() returned nil")
	}
	if got.Subject != "test-user" {
		t.Errorf("GetClaims().Subject = %v, want %v", got.Subject, "test-user")
	}
}

func TestGetClaims_Missing(t *testing.T) {
	ctx := context.Background()
	got := GetClaims(ctx)

	if got != nil {
		t.Error("GetClaims() should return nil for context without claims")
	}
}

// Integration test with real JWT verification
func TestVerifier_Verify_Integration(t *testing.T) {
	// Generate RSA key pair
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("failed to generate key: %v", err)
	}

	kid := "test-key-id"

	// Start mock JWKS server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		jwks := JWKS{
			Keys: []JWK{
				{
					Kty: "RSA",
					Kid: kid,
					Use: "sig",
					Alg: "RS256",
					N:   base64.RawURLEncoding.EncodeToString(privateKey.PublicKey.N.Bytes()),
					E:   base64.RawURLEncoding.EncodeToString(bigIntToBytes(privateKey.PublicKey.E)),
				},
			},
		}
		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(jwks); err != nil {
			t.Errorf("failed to encode JWKS: %v", err)
		}
	}))
	defer server.Close()

	// Create verifier with mock JWKS URL
	verifier := &Verifier{
		domain:   "test.auth0.com",
		audience: "https://api.test.com",
		jwks:     NewJWKSCache(server.URL),
	}

	// Create a valid token
	now := time.Now()
	claims := jwt.MapClaims{
		"iss":   "https://test.auth0.com/",
		"sub":   "auth0|123456",
		"aud":   []string{"https://api.test.com"},
		"exp":   now.Add(time.Hour).Unix(),
		"iat":   now.Unix(),
		"email": "test@example.com",
	}

	token := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
	token.Header["kid"] = kid
	tokenString, err := token.SignedString(privateKey)
	if err != nil {
		t.Fatalf("failed to sign token: %v", err)
	}

	// Verify token
	verifiedClaims, err := verifier.Verify(context.Background(), tokenString)
	if err != nil {
		t.Fatalf("Verify() error = %v", err)
	}

	if verifiedClaims.Subject != "auth0|123456" {
		t.Errorf("Subject = %v, want %v", verifiedClaims.Subject, "auth0|123456")
	}
	if verifiedClaims.Email != "test@example.com" {
		t.Errorf("Email = %v, want %v", verifiedClaims.Email, "test@example.com")
	}
}

func TestVerifier_Verify_InvalidAudience(t *testing.T) {
	privateKey, _ := rsa.GenerateKey(rand.Reader, 2048)
	kid := "test-key-id"

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		jwks := JWKS{
			Keys: []JWK{
				{
					Kty: "RSA",
					Kid: kid,
					Use: "sig",
					N:   base64.RawURLEncoding.EncodeToString(privateKey.PublicKey.N.Bytes()),
					E:   base64.RawURLEncoding.EncodeToString(bigIntToBytes(privateKey.PublicKey.E)),
				},
			},
		}
		if err := json.NewEncoder(w).Encode(jwks); err != nil {
			t.Errorf("failed to encode JWKS: %v", err)
		}
	}))
	defer server.Close()

	verifier := &Verifier{
		domain:   "test.auth0.com",
		audience: "https://api.correct.com",
		jwks:     NewJWKSCache(server.URL),
	}

	now := time.Now()
	claims := jwt.MapClaims{
		"iss": "https://test.auth0.com/",
		"sub": "auth0|123456",
		"aud": []string{"https://api.wrong.com"}, // Wrong audience
		"exp": now.Add(time.Hour).Unix(),
		"iat": now.Unix(),
	}

	token := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
	token.Header["kid"] = kid
	tokenString, _ := token.SignedString(privateKey)

	_, err := verifier.Verify(context.Background(), tokenString)
	if err == nil {
		t.Error("expected error for invalid audience")
	}
}

func TestVerifier_Verify_ExpiredToken(t *testing.T) {
	privateKey, _ := rsa.GenerateKey(rand.Reader, 2048)
	kid := "test-key-id"

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		jwks := JWKS{
			Keys: []JWK{
				{
					Kty: "RSA",
					Kid: kid,
					Use: "sig",
					N:   base64.RawURLEncoding.EncodeToString(privateKey.PublicKey.N.Bytes()),
					E:   base64.RawURLEncoding.EncodeToString(bigIntToBytes(privateKey.PublicKey.E)),
				},
			},
		}
		if err := json.NewEncoder(w).Encode(jwks); err != nil {
			t.Errorf("failed to encode JWKS: %v", err)
		}
	}))
	defer server.Close()

	verifier := &Verifier{
		domain:   "test.auth0.com",
		audience: "https://api.test.com",
		jwks:     NewJWKSCache(server.URL),
	}

	now := time.Now()
	claims := jwt.MapClaims{
		"iss": "https://test.auth0.com/",
		"sub": "auth0|123456",
		"aud": []string{"https://api.test.com"},
		"exp": now.Add(-time.Hour).Unix(), // Expired
		"iat": now.Add(-2 * time.Hour).Unix(),
	}

	token := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
	token.Header["kid"] = kid
	tokenString, _ := token.SignedString(privateKey)

	_, err := verifier.Verify(context.Background(), tokenString)
	if err == nil {
		t.Error("expected error for expired token")
	}
}

func TestVerifier_Middleware(t *testing.T) {
	privateKey, _ := rsa.GenerateKey(rand.Reader, 2048)
	kid := "test-key-id"

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		jwks := JWKS{
			Keys: []JWK{
				{
					Kty: "RSA",
					Kid: kid,
					Use: "sig",
					N:   base64.RawURLEncoding.EncodeToString(privateKey.PublicKey.N.Bytes()),
					E:   base64.RawURLEncoding.EncodeToString(bigIntToBytes(privateKey.PublicKey.E)),
				},
			},
		}
		if err := json.NewEncoder(w).Encode(jwks); err != nil {
			t.Errorf("failed to encode JWKS: %v", err)
		}
	}))
	defer server.Close()

	verifier := &Verifier{
		domain:   "test.auth0.com",
		audience: "https://api.test.com",
		jwks:     NewJWKSCache(server.URL),
	}

	// Create a valid token
	now := time.Now()
	claims := jwt.MapClaims{
		"iss": "https://test.auth0.com/",
		"sub": "auth0|123456",
		"aud": []string{"https://api.test.com"},
		"exp": now.Add(time.Hour).Unix(),
		"iat": now.Unix(),
	}
	token := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
	token.Header["kid"] = kid
	tokenString, _ := token.SignedString(privateKey)

	// Test handler that checks for claims
	var gotClaims *Claims
	handler := verifier.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotClaims = GetClaims(r.Context())
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Authorization", "Bearer "+tokenString)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", rec.Code)
	}
	if gotClaims == nil {
		t.Error("expected claims in context")
	}
	if gotClaims != nil && gotClaims.Subject != "auth0|123456" {
		t.Errorf("Subject = %v, want %v", gotClaims.Subject, "auth0|123456")
	}
}

func TestVerifier_Middleware_MissingHeader(t *testing.T) {
	verifier, _ := NewVerifier(Config{Domain: "test.auth0.com", Audience: "https://api.test.com"})

	handler := verifier.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Error("handler should not be called")
	}))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Errorf("expected status 401, got %d", rec.Code)
	}
}

// Helper to convert int to bytes for JWT e parameter
func bigIntToBytes(e int) []byte {
	return []byte(fmt.Sprintf("%c%c%c", byte(e>>16), byte(e>>8), byte(e)))
}
