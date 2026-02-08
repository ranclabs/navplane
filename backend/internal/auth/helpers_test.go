package auth

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestExtractBearerToken_MissingHeader(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/", nil)

	token, err := ExtractBearerToken(req)

	if err != ErrMissingAuthHeader {
		t.Errorf("expected ErrMissingAuthHeader, got %v", err)
	}
	if token != "" {
		t.Errorf("expected empty token, got %q", token)
	}
}

func TestExtractBearerToken_WrongScheme_Basic(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Authorization", "Basic dXNlcjpwYXNz")

	token, err := ExtractBearerToken(req)

	if err != ErrInvalidAuthScheme {
		t.Errorf("expected ErrInvalidAuthScheme, got %v", err)
	}
	if token != "" {
		t.Errorf("expected empty token, got %q", token)
	}
}

func TestExtractBearerToken_WrongScheme_ApiKey(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Authorization", "ApiKey sk-1234567890")

	token, err := ExtractBearerToken(req)

	if err != ErrInvalidAuthScheme {
		t.Errorf("expected ErrInvalidAuthScheme, got %v", err)
	}
	if token != "" {
		t.Errorf("expected empty token, got %q", token)
	}
}

func TestExtractBearerToken_BearerWithEmptyToken(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Authorization", "Bearer ")

	token, err := ExtractBearerToken(req)

	if err != ErrEmptyToken {
		t.Errorf("expected ErrEmptyToken, got %v", err)
	}
	if token != "" {
		t.Errorf("expected empty token, got %q", token)
	}
}

func TestExtractBearerToken_BearerNoSpace(t *testing.T) {
	// "Bearer" without trailing space should fail
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Authorization", "Bearer")

	token, err := ExtractBearerToken(req)

	if err != ErrInvalidAuthScheme {
		t.Errorf("expected ErrInvalidAuthScheme, got %v", err)
	}
	if token != "" {
		t.Errorf("expected empty token, got %q", token)
	}
}

func TestExtractBearerToken_ValidToken(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Authorization", "Bearer np-test-token-12345")

	token, err := ExtractBearerToken(req)

	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}
	if token != "np-test-token-12345" {
		t.Errorf("expected token 'np-test-token-12345', got %q", token)
	}
}

func TestExtractBearerToken_ValidTokenWithSpecialChars(t *testing.T) {
	// Tokens may contain various characters
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Authorization", "Bearer sk-proj_abc123-XYZ_789")

	token, err := ExtractBearerToken(req)

	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}
	if token != "sk-proj_abc123-XYZ_789" {
		t.Errorf("expected token 'sk-proj_abc123-XYZ_789', got %q", token)
	}
}

func TestExtractBearerToken_CaseSensitiveScheme(t *testing.T) {
	// "bearer" lowercase should fail (RFC 6750 uses "Bearer")
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Authorization", "bearer some-token")

	token, err := ExtractBearerToken(req)

	if err != ErrInvalidAuthScheme {
		t.Errorf("expected ErrInvalidAuthScheme, got %v", err)
	}
	if token != "" {
		t.Errorf("expected empty token, got %q", token)
	}
}

// ========================================================================
// WriteJSONError Tests
// ========================================================================

func TestWriteJSONError_SetsContentType(t *testing.T) {
	rec := httptest.NewRecorder()

	WriteJSONError(rec, http.StatusBadRequest, "bad request", "invalid_request_error")

	contentType := rec.Header().Get("Content-Type")
	if contentType != "application/json" {
		t.Errorf("expected Content-Type 'application/json', got %q", contentType)
	}
}

func TestWriteJSONError_SetsStatusCode(t *testing.T) {
	rec := httptest.NewRecorder()

	WriteJSONError(rec, http.StatusTeapot, "i'm a teapot", "teapot_error")

	if rec.Code != http.StatusTeapot {
		t.Errorf("expected status 418, got %d", rec.Code)
	}
}

func TestWriteJSONError_WritesOpenAIStyleJSON(t *testing.T) {
	rec := httptest.NewRecorder()

	WriteJSONError(rec, http.StatusBadRequest, "something went wrong", "invalid_request_error")

	var resp APIError
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to parse JSON response: %v", err)
	}

	if resp.Error.Message != "something went wrong" {
		t.Errorf("expected message 'something went wrong', got %q", resp.Error.Message)
	}
	if resp.Error.Type != "invalid_request_error" {
		t.Errorf("expected type 'invalid_request_error', got %q", resp.Error.Type)
	}
}

// ========================================================================
// WriteUnauthorized Tests
// ========================================================================

func TestWriteUnauthorized_Returns401(t *testing.T) {
	rec := httptest.NewRecorder()

	WriteUnauthorized(rec)

	if rec.Code != http.StatusUnauthorized {
		t.Errorf("expected status 401, got %d", rec.Code)
	}
}

func TestWriteUnauthorized_SetsContentType(t *testing.T) {
	rec := httptest.NewRecorder()

	WriteUnauthorized(rec)

	contentType := rec.Header().Get("Content-Type")
	if contentType != "application/json" {
		t.Errorf("expected Content-Type 'application/json', got %q", contentType)
	}
}

func TestWriteUnauthorized_WritesCorrectJSON(t *testing.T) {
	rec := httptest.NewRecorder()

	WriteUnauthorized(rec)

	var resp APIError
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to parse JSON response: %v", err)
	}

	if resp.Error.Message != "unauthorized" {
		t.Errorf("expected message 'unauthorized', got %q", resp.Error.Message)
	}
	if resp.Error.Type != "authentication_error" {
		t.Errorf("expected type 'authentication_error', got %q", resp.Error.Type)
	}
}

// ========================================================================
// WriteForbidden Tests
// ========================================================================

func TestWriteForbidden_Returns403(t *testing.T) {
	rec := httptest.NewRecorder()

	WriteForbidden(rec)

	if rec.Code != http.StatusForbidden {
		t.Errorf("expected status 403, got %d", rec.Code)
	}
}

func TestWriteForbidden_SetsContentType(t *testing.T) {
	rec := httptest.NewRecorder()

	WriteForbidden(rec)

	contentType := rec.Header().Get("Content-Type")
	if contentType != "application/json" {
		t.Errorf("expected Content-Type 'application/json', got %q", contentType)
	}
}

func TestWriteForbidden_WritesCorrectJSON(t *testing.T) {
	rec := httptest.NewRecorder()

	WriteForbidden(rec)

	var resp APIError
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to parse JSON response: %v", err)
	}

	if resp.Error.Message != "forbidden" {
		t.Errorf("expected message 'forbidden', got %q", resp.Error.Message)
	}
	if resp.Error.Type != "authentication_error" {
		t.Errorf("expected type 'authentication_error', got %q", resp.Error.Type)
	}
}

// ========================================================================
// Integration-style test: simulating how middleware would use these
// ========================================================================

func TestAuthHelpers_IntegrationScenario(t *testing.T) {
	// Simulates how middleware will use these helpers

	tests := []struct {
		name            string
		authHeader      string
		expectedStatus  int
		expectedMessage string
		expectedType    string
	}{
		{
			name:            "missing header",
			authHeader:      "",
			expectedStatus:  http.StatusUnauthorized,
			expectedMessage: "unauthorized",
			expectedType:    "authentication_error",
		},
		{
			name:            "basic auth scheme",
			authHeader:      "Basic dXNlcjpwYXNz",
			expectedStatus:  http.StatusUnauthorized,
			expectedMessage: "unauthorized",
			expectedType:    "authentication_error",
		},
		{
			name:            "bearer with empty token",
			authHeader:      "Bearer ",
			expectedStatus:  http.StatusUnauthorized,
			expectedMessage: "unauthorized",
			expectedType:    "authentication_error",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodPost, "/v1/chat/completions", nil)
			if tt.authHeader != "" {
				req.Header.Set("Authorization", tt.authHeader)
			}
			rec := httptest.NewRecorder()

			// Simulate middleware logic
			_, err := ExtractBearerToken(req)
			if err != nil {
				WriteUnauthorized(rec)
			}

			// Verify response
			if rec.Code != tt.expectedStatus {
				t.Errorf("expected status %d, got %d", tt.expectedStatus, rec.Code)
			}

			var resp APIError
			if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
				t.Fatalf("failed to parse JSON: %v", err)
			}
			if resp.Error.Message != tt.expectedMessage {
				t.Errorf("expected message %q, got %q", tt.expectedMessage, resp.Error.Message)
			}
			if resp.Error.Type != tt.expectedType {
				t.Errorf("expected type %q, got %q", tt.expectedType, resp.Error.Type)
			}

			contentType := rec.Header().Get("Content-Type")
			if contentType != "application/json" {
				t.Errorf("expected Content-Type 'application/json', got %q", contentType)
			}
		})
	}
}
