package jwtauth

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// Claims represents the JWT claims from Auth0.
type Claims struct {
	jwt.RegisteredClaims
	Email         string   `json:"email,omitempty"`
	EmailVerified bool     `json:"email_verified,omitempty"`
	Name          string   `json:"name,omitempty"`
	Permissions   []string `json:"permissions,omitempty"`
}

// Verifier handles JWT verification using Auth0.
type Verifier struct {
	domain   string
	audience string
	jwks     *JWKSCache
}

// Config holds Auth0 JWT verification configuration.
type Config struct {
	Domain   string // e.g., "your-tenant.auth0.com"
	Audience string // e.g., "https://api.navplane.io"
}

// NewVerifier creates a new JWT verifier.
func NewVerifier(cfg Config) (*Verifier, error) {
	if cfg.Domain == "" {
		return nil, errors.New("domain is required")
	}
	if cfg.Audience == "" {
		return nil, errors.New("audience is required")
	}

	// Remove protocol if present
	domain := strings.TrimPrefix(cfg.Domain, "https://")
	domain = strings.TrimPrefix(domain, "http://")

	return &Verifier{
		domain:   domain,
		audience: cfg.Audience,
		jwks:     NewJWKSCache(fmt.Sprintf("https://%s/.well-known/jwks.json", domain)),
	}, nil
}

// Verify verifies a JWT token and returns the claims.
func (v *Verifier) Verify(ctx context.Context, tokenString string) (*Claims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(token *jwt.Token) (any, error) {
		// Verify signing method
		if _, ok := token.Method.(*jwt.SigningMethodRSA); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}

		// Get key ID from token header
		kid, ok := token.Header["kid"].(string)
		if !ok {
			return nil, errors.New("missing kid in token header")
		}

		// Get the public key from JWKS
		return v.jwks.GetKey(ctx, kid)
	})

	if err != nil {
		return nil, fmt.Errorf("failed to parse token: %w", err)
	}

	claims, ok := token.Claims.(*Claims)
	if !ok || !token.Valid {
		return nil, errors.New("invalid token")
	}

	// Verify audience
	if !v.verifyAudience(claims) {
		return nil, errors.New("invalid audience")
	}

	// Verify issuer
	expectedIssuer := fmt.Sprintf("https://%s/", v.domain)
	if claims.Issuer != expectedIssuer {
		return nil, fmt.Errorf("invalid issuer: expected %s, got %s", expectedIssuer, claims.Issuer)
	}

	return claims, nil
}

func (v *Verifier) verifyAudience(claims *Claims) bool {
	for _, aud := range claims.Audience {
		if aud == v.audience {
			return true
		}
	}
	return false
}

// Auth0UserID returns the user ID from the claims (subject).
func (c *Claims) Auth0UserID() string {
	return c.Subject
}

// HasPermission checks if the user has a specific permission.
func (c *Claims) HasPermission(perm string) bool {
	for _, p := range c.Permissions {
		if p == perm {
			return true
		}
	}
	return false
}

// Middleware creates HTTP middleware that verifies JWT tokens.
func (v *Verifier) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		authHeader := r.Header.Get("Authorization")
		if authHeader == "" {
			writeAuthError(w, http.StatusUnauthorized, "missing authorization header")
			return
		}

		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
			writeAuthError(w, http.StatusUnauthorized, "invalid authorization header format")
			return
		}

		tokenString := parts[1]
		claims, err := v.Verify(r.Context(), tokenString)
		if err != nil {
			log.Printf("JWT verification failed: %v", err)
			writeAuthError(w, http.StatusUnauthorized, "invalid token")
			return
		}

		// Add claims to context
		ctx := context.WithValue(r.Context(), ClaimsContextKey, claims)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

type contextKey string

const ClaimsContextKey contextKey = "jwtclaims"

// GetClaims retrieves JWT claims from the request context.
func GetClaims(ctx context.Context) *Claims {
	claims, _ := ctx.Value(ClaimsContextKey).(*Claims)
	return claims
}

func writeAuthError(w http.ResponseWriter, status int, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	fmt.Fprintf(w, `{"error":{"message":%q,"type":"authentication_error"}}`, message)
}

// JWKSCache caches JWKS keys from Auth0.
type JWKSCache struct {
	url        string
	mu         sync.RWMutex
	keys       map[string]any // kid -> public key
	lastFetch  time.Time
	cacheTTL   time.Duration
	httpClient *http.Client
}

// NewJWKSCache creates a new JWKS cache.
func NewJWKSCache(jwksURL string) *JWKSCache {
	return &JWKSCache{
		url:      jwksURL,
		keys:     make(map[string]any),
		cacheTTL: 10 * time.Minute,
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

// GetKey returns the public key for the given key ID.
func (c *JWKSCache) GetKey(ctx context.Context, kid string) (any, error) {
	c.mu.RLock()
	key, ok := c.keys[kid]
	needsRefresh := time.Since(c.lastFetch) > c.cacheTTL
	c.mu.RUnlock()

	if ok && !needsRefresh {
		return key, nil
	}

	// Fetch new keys
	if err := c.refresh(ctx); err != nil {
		// If we have a cached key and refresh fails, use the cached key
		if ok {
			log.Printf("JWKS refresh failed, using cached key: %v", err)
			return key, nil
		}
		return nil, fmt.Errorf("failed to fetch JWKS: %w", err)
	}

	c.mu.RLock()
	key, ok = c.keys[kid]
	c.mu.RUnlock()

	if !ok {
		return nil, fmt.Errorf("key %s not found in JWKS", kid)
	}

	return key, nil
}

// JWKS represents a JSON Web Key Set.
type JWKS struct {
	Keys []JWK `json:"keys"`
}

// JWK represents a JSON Web Key.
type JWK struct {
	Kty string `json:"kty"`
	Kid string `json:"kid"`
	Use string `json:"use"`
	N   string `json:"n"`
	E   string `json:"e"`
	Alg string `json:"alg"`
}

func (c *JWKSCache) refresh(ctx context.Context) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	// Double-check after acquiring lock
	if time.Since(c.lastFetch) < c.cacheTTL && len(c.keys) > 0 {
		return nil
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.url, nil)
	if err != nil {
		return err
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("JWKS endpoint returned status %d", resp.StatusCode)
	}

	var jwks JWKS
	if err := decodeJSON(resp.Body, &jwks); err != nil {
		return fmt.Errorf("failed to decode JWKS: %w", err)
	}

	newKeys := make(map[string]any)
	for _, key := range jwks.Keys {
		if key.Kty != "RSA" || key.Use != "sig" {
			continue
		}

		publicKey, err := parseRSAPublicKey(key.N, key.E)
		if err != nil {
			log.Printf("failed to parse RSA key %s: %v", key.Kid, err)
			continue
		}

		newKeys[key.Kid] = publicKey
	}

	c.keys = newKeys
	c.lastFetch = time.Now()

	return nil
}
