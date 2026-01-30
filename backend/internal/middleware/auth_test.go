package middleware

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"navplane/internal/auth"
)

// mockAuthStore implements auth.AuthStore for testing.
type mockAuthStore struct {
	validateFunc func(ctx context.Context, token string) (*auth.OrgContext, error)
}

func (m *mockAuthStore) ValidateToken(ctx context.Context, token string) (*auth.OrgContext, error) {
	if m.validateFunc != nil {
		return m.validateFunc(ctx, token)
	}
	return nil, auth.ErrTokenNotFound
}

func (m *mockAuthStore) Close() error {
	return nil
}

// testHandler is a simple handler that returns 200 OK.
func testHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Check if org context is available
		orgCtx, ok := GetOrgContext(r.Context())
		if !ok {
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(map[string]string{"error": "org context not found"})
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]string{
			"org_id":   orgCtx.OrgID,
			"org_name": orgCtx.OrgName,
		})
	})
}

func TestRequireAuth_MissingAuthorizationHeader(t *testing.T) {
	store := &mockAuthStore{}
	middleware := RequireAuth(store)
	handler := middleware(testHandler())

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Errorf("expected status 401, got %d", rec.Code)
	}

	// Verify JSON error response
	var resp auth.APIError
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}
	if resp.Error.Message != "unauthorized" {
		t.Errorf("expected message 'unauthorized', got %q", resp.Error.Message)
	}
}

func TestRequireAuth_InvalidScheme(t *testing.T) {
	store := &mockAuthStore{}
	middleware := RequireAuth(store)
	handler := middleware(testHandler())

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.Header.Set("Authorization", "Basic dXNlcjpwYXNz")
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Errorf("expected status 401, got %d", rec.Code)
	}
}

func TestRequireAuth_EmptyToken(t *testing.T) {
	store := &mockAuthStore{}
	middleware := RequireAuth(store)
	handler := middleware(testHandler())

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.Header.Set("Authorization", "Bearer ")
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Errorf("expected status 401, got %d", rec.Code)
	}
}

func TestRequireAuth_InvalidToken(t *testing.T) {
	store := &mockAuthStore{
		validateFunc: func(ctx context.Context, token string) (*auth.OrgContext, error) {
			return nil, auth.ErrTokenNotFound
		},
	}
	middleware := RequireAuth(store)
	handler := middleware(testHandler())

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.Header.Set("Authorization", "Bearer invalid-token")
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Errorf("expected status 403, got %d", rec.Code)
	}

	// Verify JSON error response
	var resp auth.APIError
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}
	if resp.Error.Message != "forbidden" {
		t.Errorf("expected message 'forbidden', got %q", resp.Error.Message)
	}
}

func TestRequireAuth_DisabledOrg(t *testing.T) {
	store := &mockAuthStore{
		validateFunc: func(ctx context.Context, token string) (*auth.OrgContext, error) {
			return &auth.OrgContext{
				OrgID:   "org-123",
				OrgName: "Disabled Org",
				Enabled: false, // Org is disabled
			}, nil
		},
	}
	middleware := RequireAuth(store)
	handler := middleware(testHandler())

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.Header.Set("Authorization", "Bearer valid-token")
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Errorf("expected status 403 for disabled org, got %d", rec.Code)
	}
}

func TestRequireAuth_ValidToken(t *testing.T) {
	store := &mockAuthStore{
		validateFunc: func(ctx context.Context, token string) (*auth.OrgContext, error) {
			if token == "valid-token-123" {
				return &auth.OrgContext{
					OrgID:   "org-456",
					OrgName: "Test Organization",
					Enabled: true,
				}, nil
			}
			return nil, auth.ErrTokenNotFound
		},
	}
	middleware := RequireAuth(store)
	handler := middleware(testHandler())

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.Header.Set("Authorization", "Bearer valid-token-123")
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", rec.Code)
	}

	// Verify handler received org context
	var resp map[string]string
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}
	if resp["org_id"] != "org-456" {
		t.Errorf("expected org_id 'org-456', got %q", resp["org_id"])
	}
	if resp["org_name"] != "Test Organization" {
		t.Errorf("expected org_name 'Test Organization', got %q", resp["org_name"])
	}
}

func TestGetOrgContext_NotSet(t *testing.T) {
	ctx := context.Background()
	orgCtx, ok := GetOrgContext(ctx)

	if ok {
		t.Error("expected ok=false when org context not set")
	}
	if orgCtx != nil {
		t.Error("expected nil org context when not set")
	}
}

func TestGetOrgContext_Set(t *testing.T) {
	expected := &auth.OrgContext{
		OrgID:   "org-789",
		OrgName: "My Org",
		Enabled: true,
	}

	ctx := context.WithValue(context.Background(), OrgContextKey, expected)
	orgCtx, ok := GetOrgContext(ctx)

	if !ok {
		t.Error("expected ok=true when org context is set")
	}
	if orgCtx != expected {
		t.Errorf("expected org context %+v, got %+v", expected, orgCtx)
	}
}

func TestRequireAuth_ServerError(t *testing.T) {
	store := &mockAuthStore{
		validateFunc: func(ctx context.Context, token string) (*auth.OrgContext, error) {
			return nil, errors.New("database connection failed")
		},
	}
	middleware := RequireAuth(store)
	handler := middleware(testHandler())

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.Header.Set("Authorization", "Bearer some-token")
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Errorf("expected status 500, got %d", rec.Code)
	}

	// Verify JSON error response
	var resp auth.APIError
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}
	if resp.Error.Message != "internal error" {
		t.Errorf("expected message 'internal error', got %q", resp.Error.Message)
	}
	if resp.Error.Type != "server_error" {
		t.Errorf("expected type 'server_error', got %q", resp.Error.Type)
	}
}

func TestRequireAuth_ContentTypeAlwaysJSON(t *testing.T) {
	// All error responses should have Content-Type: application/json
	tests := []struct {
		name       string
		authHeader string
		store      *mockAuthStore
	}{
		{
			name:       "missing auth",
			authHeader: "",
			store:      &mockAuthStore{},
		},
		{
			name:       "invalid scheme",
			authHeader: "Basic xyz",
			store:      &mockAuthStore{},
		},
		{
			name:       "invalid token",
			authHeader: "Bearer bad-token",
			store: &mockAuthStore{
				validateFunc: func(ctx context.Context, token string) (*auth.OrgContext, error) {
					return nil, auth.ErrTokenNotFound
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			middleware := RequireAuth(tt.store)
			handler := middleware(testHandler())

			req := httptest.NewRequest(http.MethodGet, "/test", nil)
			if tt.authHeader != "" {
				req.Header.Set("Authorization", tt.authHeader)
			}
			rec := httptest.NewRecorder()

			handler.ServeHTTP(rec, req)

			contentType := rec.Header().Get("Content-Type")
			if contentType != "application/json" {
				t.Errorf("expected Content-Type 'application/json', got %q", contentType)
			}
		})
	}
}
