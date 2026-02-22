package middleware

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"lectr/internal/org"
)

func TestExtractBearerToken(t *testing.T) {
	tests := []struct {
		name        string
		authHeader  string
		wantToken   string
		wantErr     bool
	}{
		{
			name:       "valid bearer token",
			authHeader: "Bearer lc_abc123",
			wantToken:  "lc_abc123",
			wantErr:    false,
		},
		{
			name:       "missing header",
			authHeader: "",
			wantErr:    true,
		},
		{
			name:       "wrong scheme - Basic",
			authHeader: "Basic abc123",
			wantErr:    true,
		},
		{
			name:       "wrong scheme - ApiKey",
			authHeader: "ApiKey abc123",
			wantErr:    true,
		},
		{
			name:       "empty token",
			authHeader: "Bearer ",
			wantErr:    true,
		},
		{
			name:       "bearer without space",
			authHeader: "Bearerabc123",
			wantErr:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/", nil)
			if tt.authHeader != "" {
				req.Header.Set("Authorization", tt.authHeader)
			}

			token, err := extractBearerToken(req)
			if tt.wantErr {
				if err == nil {
					t.Error("expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}

			if token != tt.wantToken {
				t.Errorf("expected token %q, got %q", tt.wantToken, token)
			}
		})
	}
}

func TestGetOrg(t *testing.T) {
	t.Run("org in context", func(t *testing.T) {
		testOrg := &org.Org{Name: "Test Org"}
		ctx := context.WithValue(context.Background(), OrgContextKey, testOrg)

		got := GetOrg(ctx)
		if got == nil {
			t.Fatal("expected org, got nil")
		}
		if got.Name != "Test Org" {
			t.Errorf("expected name 'Test Org', got %q", got.Name)
		}
	})

	t.Run("no org in context", func(t *testing.T) {
		got := GetOrg(context.Background())
		if got != nil {
			t.Errorf("expected nil, got %v", got)
		}
	})

	t.Run("wrong type in context", func(t *testing.T) {
		ctx := context.WithValue(context.Background(), OrgContextKey, "not an org")
		got := GetOrg(ctx)
		if got != nil {
			t.Errorf("expected nil for wrong type, got %v", got)
		}
	})
}

func TestWriteAuthError(t *testing.T) {
	tests := []struct {
		name       string
		status     int
		message    string
		wantStatus int
		wantBody   string
	}{
		{
			name:       "unauthorized",
			status:     http.StatusUnauthorized,
			message:    "missing API key",
			wantStatus: http.StatusUnauthorized,
			wantBody:   `{"error":{"message":"missing API key","type":"authentication_error"}}`,
		},
		{
			name:       "forbidden",
			status:     http.StatusForbidden,
			message:    "organization disabled",
			wantStatus: http.StatusForbidden,
			wantBody:   `{"error":{"message":"organization disabled","type":"authentication_error"}}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			writeAuthError(w, tt.status, tt.message)

			if w.Code != tt.wantStatus {
				t.Errorf("expected status %d, got %d", tt.wantStatus, w.Code)
			}

			if w.Header().Get("Content-Type") != "application/json" {
				t.Errorf("expected Content-Type application/json, got %s", w.Header().Get("Content-Type"))
			}

			if w.Body.String() != tt.wantBody {
				t.Errorf("expected body %q, got %q", tt.wantBody, w.Body.String())
			}
		})
	}
}
