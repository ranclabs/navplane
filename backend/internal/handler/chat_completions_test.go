package handler

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"navplane/internal/config"
)

// roundTripperFunc allows using a function as an http.RoundTripper for testing.
type roundTripperFunc func(*http.Request) (*http.Response, error)

func (f roundTripperFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}

// testConfig returns a config suitable for testing
func testConfig() *config.Config {
	return &config.Config{
		Port:        "8080",
		Environment: "development",
		Provider: config.ProviderConfig{
			BaseURL: "https://api.openai.com",
			APIKey:  "sk-test-key-for-testing-only",
		},
	}
}

// mockHTTPClient creates an HTTP client that calls the provided handler for all requests
func mockHTTPClient(handler func(req *http.Request) (*http.Response, error)) *http.Client {
	return &http.Client{
		Transport: roundTripperFunc(handler),
	}
}

// --- Method Not Allowed Tests ---
// These test NavPlane's routing logic, not upstream behavior

func TestChatCompletions_MethodNotAllowed_GET(t *testing.T) {
	cfg := testConfig()
	client := mockHTTPClient(func(req *http.Request) (*http.Response, error) {
		t.Fatal("upstream should not be called for GET request")
		return nil, nil
	})

	handler := NewChatCompletionsHandlerWithClient(cfg, client)
	req := httptest.NewRequest(http.MethodGet, "/v1/chat/completions", nil)
	rec := httptest.NewRecorder()

	handler(rec, req)

	if rec.Code != http.StatusMethodNotAllowed {
		t.Errorf("expected status 405, got %d", rec.Code)
	}

	assertJSONError(t, rec.Body.Bytes(), "method not allowed", "invalid_request_error")
}

func TestChatCompletions_MethodNotAllowed_PUT(t *testing.T) {
	cfg := testConfig()
	client := mockHTTPClient(func(req *http.Request) (*http.Response, error) {
		t.Fatal("upstream should not be called for PUT request")
		return nil, nil
	})

	handler := NewChatCompletionsHandlerWithClient(cfg, client)
	body := `{"model": "gpt-4", "messages": [{"role": "user", "content": "Hello"}]}`
	req := httptest.NewRequest(http.MethodPut, "/v1/chat/completions", bytes.NewBufferString(body))
	rec := httptest.NewRecorder()

	handler(rec, req)

	if rec.Code != http.StatusMethodNotAllowed {
		t.Errorf("expected status 405, got %d", rec.Code)
	}
}

func TestChatCompletions_MethodNotAllowed_DELETE(t *testing.T) {
	cfg := testConfig()
	client := mockHTTPClient(func(req *http.Request) (*http.Response, error) {
		t.Fatal("upstream should not be called for DELETE request")
		return nil, nil
	})

	handler := NewChatCompletionsHandlerWithClient(cfg, client)
	req := httptest.NewRequest(http.MethodDelete, "/v1/chat/completions", nil)
	rec := httptest.NewRecorder()

	handler(rec, req)

	if rec.Code != http.StatusMethodNotAllowed {
		t.Errorf("expected status 405, got %d", rec.Code)
	}
}

// --- Passthrough Tests ---
// These test that requests are forwarded and responses are returned as-is

func TestChatCompletions_PassthroughSuccess(t *testing.T) {
	cfg := testConfig()

	// Mock upstream response (what OpenAI would return)
	upstreamResponse := map[string]any{
		"id":      "chatcmpl-123",
		"object":  "chat.completion",
		"created": 1677652288,
		"model":   "gpt-4",
		"choices": []map[string]any{
			{
				"index": 0,
				"message": map[string]any{
					"role":    "assistant",
					"content": "Hello! How can I help you today?",
				},
				"finish_reason": "stop",
			},
		},
	}
	upstreamBody, _ := json.Marshal(upstreamResponse)

	client := mockHTTPClient(func(req *http.Request) (*http.Response, error) {
		// Verify request was forwarded correctly
		if req.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", req.Method)
		}
		if !strings.HasSuffix(req.URL.String(), "/v1/chat/completions") {
			t.Errorf("unexpected URL: %s", req.URL.String())
		}

		return &http.Response{
			StatusCode: http.StatusOK,
			Header: http.Header{
				"Content-Type": []string{"application/json"},
				"X-Request-Id": []string{"req-123"},
			},
			Body: io.NopCloser(bytes.NewReader(upstreamBody)),
		}, nil
	})

	handler := NewChatCompletionsHandlerWithClient(cfg, client)

	body := `{"model": "gpt-4", "messages": [{"role": "user", "content": "Hello!"}]}`
	req := httptest.NewRequest(http.MethodPost, "/v1/chat/completions", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	handler(rec, req)

	// Verify status code passthrough
	if rec.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", rec.Code)
	}

	// Verify Content-Type passthrough
	if ct := rec.Header().Get("Content-Type"); ct != "application/json" {
		t.Errorf("expected Content-Type 'application/json', got '%s'", ct)
	}

	// Verify X-Request-Id passthrough
	if rid := rec.Header().Get("X-Request-Id"); rid != "req-123" {
		t.Errorf("expected X-Request-Id 'req-123', got '%s'", rid)
	}

	// Verify response body passthrough
	var resp map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}
	if resp["id"] != "chatcmpl-123" {
		t.Errorf("expected id 'chatcmpl-123', got %v", resp["id"])
	}
}

func TestChatCompletions_PassthroughUpstreamError(t *testing.T) {
	cfg := testConfig()

	// Mock upstream error response (what OpenAI would return for invalid model)
	upstreamError := map[string]any{
		"error": map[string]any{
			"message": "The model `gpt-5` does not exist",
			"type":    "invalid_request_error",
			"param":   "model",
			"code":    "model_not_found",
		},
	}
	upstreamBody, _ := json.Marshal(upstreamError)

	client := mockHTTPClient(func(req *http.Request) (*http.Response, error) {
		return &http.Response{
			StatusCode: http.StatusNotFound,
			Header: http.Header{
				"Content-Type": []string{"application/json"},
			},
			Body: io.NopCloser(bytes.NewReader(upstreamBody)),
		}, nil
	})

	handler := NewChatCompletionsHandlerWithClient(cfg, client)

	// Send request with invalid model - we don't validate, upstream does
	body := `{"model": "gpt-5", "messages": [{"role": "user", "content": "Hello!"}]}`
	req := httptest.NewRequest(http.MethodPost, "/v1/chat/completions", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	handler(rec, req)

	// Verify upstream error is passed through as-is
	if rec.Code != http.StatusNotFound {
		t.Errorf("expected status 404, got %d", rec.Code)
	}

	// Verify error message is from upstream, not from us
	var resp map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}
	errObj := resp["error"].(map[string]any)
	if errObj["code"] != "model_not_found" {
		t.Errorf("expected error code 'model_not_found', got %v", errObj["code"])
	}
}

func TestChatCompletions_PassthroughValidationError(t *testing.T) {
	cfg := testConfig()

	// Mock upstream validation error (missing messages)
	upstreamError := map[string]any{
		"error": map[string]any{
			"message": "'messages' is a required property",
			"type":    "invalid_request_error",
			"param":   "messages",
		},
	}
	upstreamBody, _ := json.Marshal(upstreamError)

	client := mockHTTPClient(func(req *http.Request) (*http.Response, error) {
		return &http.Response{
			StatusCode: http.StatusBadRequest,
			Header: http.Header{
				"Content-Type": []string{"application/json"},
			},
			Body: io.NopCloser(bytes.NewReader(upstreamBody)),
		}, nil
	})

	handler := NewChatCompletionsHandlerWithClient(cfg, client)

	// Send invalid request - we pass through, upstream validates
	body := `{"model": "gpt-4"}`
	req := httptest.NewRequest(http.MethodPost, "/v1/chat/completions", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	handler(rec, req)

	// Verify upstream error is passed through
	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected status 400, got %d", rec.Code)
	}

	var resp map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}
	errObj := resp["error"].(map[string]any)
	// Verify it's the upstream's error message, not ours
	msg := errObj["message"].(string)
	if !strings.Contains(msg, "'messages' is a required property") {
		t.Errorf("expected upstream error message, got %s", msg)
	}
}

func TestChatCompletions_PassthroughInvalidJSON(t *testing.T) {
	cfg := testConfig()

	// Mock upstream error for invalid JSON
	upstreamError := map[string]any{
		"error": map[string]any{
			"message": "Could not parse JSON body",
			"type":    "invalid_request_error",
		},
	}
	upstreamBody, _ := json.Marshal(upstreamError)

	client := mockHTTPClient(func(req *http.Request) (*http.Response, error) {
		return &http.Response{
			StatusCode: http.StatusBadRequest,
			Header: http.Header{
				"Content-Type": []string{"application/json"},
			},
			Body: io.NopCloser(bytes.NewReader(upstreamBody)),
		}, nil
	})

	handler := NewChatCompletionsHandlerWithClient(cfg, client)

	// Send malformed JSON - upstream will handle the error
	body := `{"model": "gpt-4", "messages": [`
	req := httptest.NewRequest(http.MethodPost, "/v1/chat/completions", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	handler(rec, req)

	// Verify upstream error is passed through
	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected status 400, got %d", rec.Code)
	}
}

func TestChatCompletions_PassthroughRateLimitHeaders(t *testing.T) {
	cfg := testConfig()

	client := mockHTTPClient(func(req *http.Request) (*http.Response, error) {
		// Use http.Header.Set() for proper canonicalization
		headers := make(http.Header)
		headers.Set("Content-Type", "application/json")
		headers.Set("X-RateLimit-Limit-Requests", "10000")
		headers.Set("X-RateLimit-Remaining-Tokens", "9999")
		headers.Set("X-RateLimit-Reset-Requests", "1s")

		return &http.Response{
			StatusCode: http.StatusOK,
			Header:     headers,
			Body:       io.NopCloser(bytes.NewReader([]byte(`{"id":"test"}`))),
		}, nil
	})

	handler := NewChatCompletionsHandlerWithClient(cfg, client)

	body := `{"model": "gpt-4", "messages": [{"role": "user", "content": "Hi"}]}`
	req := httptest.NewRequest(http.MethodPost, "/v1/chat/completions", bytes.NewBufferString(body))
	rec := httptest.NewRecorder()

	handler(rec, req)

	// Verify rate limit headers are passed through
	if v := rec.Header().Get("X-RateLimit-Limit-Requests"); v != "10000" {
		t.Errorf("expected X-RateLimit-Limit-Requests '10000', got '%s'", v)
	}
	if v := rec.Header().Get("X-RateLimit-Remaining-Tokens"); v != "9999" {
		t.Errorf("expected X-RateLimit-Remaining-Tokens '9999', got '%s'", v)
	}
}

// --- Security Tests ---
// These verify that client credentials are never leaked to upstream

func TestChatCompletions_ClientAuthNeverForwarded(t *testing.T) {
	cfg := testConfig()

	var capturedAuthHeader string
	client := mockHTTPClient(func(req *http.Request) (*http.Response, error) {
		capturedAuthHeader = req.Header.Get("Authorization")
		return &http.Response{
			StatusCode: http.StatusOK,
			Header:     http.Header{"Content-Type": []string{"application/json"}},
			Body:       io.NopCloser(bytes.NewReader([]byte(`{"id":"test"}`))),
		}, nil
	})

	handler := NewChatCompletionsHandlerWithClient(cfg, client)

	body := `{"model": "gpt-4", "messages": [{"role": "user", "content": "Hello"}]}`
	req := httptest.NewRequest(http.MethodPost, "/v1/chat/completions", bytes.NewBufferString(body))
	// Client sends their own auth header (this should be NavPlane org token)
	req.Header.Set("Authorization", "Bearer client-secret-token-should-not-forward")
	rec := httptest.NewRecorder()

	handler(rec, req)

	// Verify upstream received the PROVIDER key, not the client key
	expectedAuth := "Bearer " + cfg.Provider.APIKey
	if capturedAuthHeader != expectedAuth {
		t.Errorf("expected upstream to receive '%s', got '%s'", expectedAuth, capturedAuthHeader)
	}
	if strings.Contains(capturedAuthHeader, "client-secret-token") {
		t.Error("client auth token was leaked to upstream!")
	}
}

func TestChatCompletions_ProviderKeyUsed(t *testing.T) {
	cfg := testConfig()
	cfg.Provider.APIKey = "sk-provider-key-12345"

	var capturedAuthHeader string
	client := mockHTTPClient(func(req *http.Request) (*http.Response, error) {
		capturedAuthHeader = req.Header.Get("Authorization")
		return &http.Response{
			StatusCode: http.StatusOK,
			Header:     http.Header{"Content-Type": []string{"application/json"}},
			Body:       io.NopCloser(bytes.NewReader([]byte(`{"id":"test"}`))),
		}, nil
	})

	handler := NewChatCompletionsHandlerWithClient(cfg, client)

	body := `{"model": "gpt-4", "messages": [{"role": "user", "content": "Hello"}]}`
	req := httptest.NewRequest(http.MethodPost, "/v1/chat/completions", bytes.NewBufferString(body))
	rec := httptest.NewRecorder()

	handler(rec, req)

	if capturedAuthHeader != "Bearer sk-provider-key-12345" {
		t.Errorf("expected provider key in auth header, got '%s'", capturedAuthHeader)
	}
}

// --- Request Body Passthrough Tests ---
// These verify the request body is forwarded exactly as received

func TestChatCompletions_RequestBodyPassthrough(t *testing.T) {
	cfg := testConfig()

	var capturedBody []byte
	client := mockHTTPClient(func(req *http.Request) (*http.Response, error) {
		var err error
		capturedBody, err = io.ReadAll(req.Body)
		if err != nil {
			t.Fatalf("failed to read captured body: %v", err)
		}
		return &http.Response{
			StatusCode: http.StatusOK,
			Header:     http.Header{"Content-Type": []string{"application/json"}},
			Body:       io.NopCloser(bytes.NewReader([]byte(`{"id":"test"}`))),
		}, nil
	})

	handler := NewChatCompletionsHandlerWithClient(cfg, client)

	// Include unknown fields that we should passthrough
	originalBody := `{
		"model": "gpt-4",
		"messages": [{"role": "user", "content": "Hello"}],
		"temperature": 0.7,
		"custom_field": "should_passthrough",
		"response_format": {"type": "json_object"},
		"seed": 42
	}`
	req := httptest.NewRequest(http.MethodPost, "/v1/chat/completions", bytes.NewBufferString(originalBody))
	rec := httptest.NewRecorder()

	handler(rec, req)

	// Verify the exact body was forwarded
	if string(capturedBody) != originalBody {
		t.Errorf("request body was modified!\nexpected: %s\ngot: %s", originalBody, string(capturedBody))
	}
}

// --- Streaming Tests ---

func TestChatCompletions_IsStreamingRequest(t *testing.T) {
	tests := []struct {
		name     string
		body     string
		expected bool
	}{
		{"stream true", `{"stream": true}`, true},
		{"stream false", `{"stream": false}`, false},
		{"stream missing", `{"model": "gpt-4"}`, false},
		{"stream null", `{"stream": null}`, false},
		{"invalid json", `{invalid}`, false},
		{"empty body", ``, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isStreamingRequest([]byte(tt.body))
			if result != tt.expected {
				t.Errorf("isStreamingRequest(%q) = %v, want %v", tt.body, result, tt.expected)
			}
		})
	}
}

func TestChatCompletions_StreamingNotImplemented(t *testing.T) {
	cfg := testConfig()
	client := mockHTTPClient(func(req *http.Request) (*http.Response, error) {
		t.Fatal("upstream should not be called for streaming request")
		return nil, nil
	})

	handler := NewChatCompletionsHandlerWithClient(cfg, client)

	body := `{"model": "gpt-4", "messages": [{"role": "user", "content": "Hi"}], "stream": true}`
	req := httptest.NewRequest(http.MethodPost, "/v1/chat/completions", bytes.NewBufferString(body))
	rec := httptest.NewRecorder()

	handler(rec, req)

	if rec.Code != http.StatusNotImplemented {
		t.Errorf("expected status 501, got %d", rec.Code)
	}

	assertJSONError(t, rec.Body.Bytes(), "streaming not implemented yet", "not_implemented_error")
}

// --- Error Handling Tests ---

func TestChatCompletions_RequestBodyTooLarge(t *testing.T) {
	cfg := testConfig()
	client := mockHTTPClient(func(req *http.Request) (*http.Response, error) {
		t.Fatal("upstream should not be called for oversized request")
		return nil, nil
	})

	handler := NewChatCompletionsHandlerWithClient(cfg, client)

	// Create a body larger than 10MB limit
	largeBody := strings.Repeat("x", 11*1024*1024)
	req := httptest.NewRequest(http.MethodPost, "/v1/chat/completions", strings.NewReader(largeBody))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	handler(rec, req)

	if rec.Code != http.StatusRequestEntityTooLarge {
		t.Errorf("expected status 413, got %d", rec.Code)
	}

	assertJSONError(t, rec.Body.Bytes(), "request body too large", "invalid_request_error")
}

func TestChatCompletions_UpstreamUnreachable(t *testing.T) {
	cfg := testConfig()

	client := mockHTTPClient(func(req *http.Request) (*http.Response, error) {
		return nil, &mockNetworkError{message: "connection refused"}
	})

	handler := NewChatCompletionsHandlerWithClient(cfg, client)

	body := `{"model": "gpt-4", "messages": [{"role": "user", "content": "Hello"}]}`
	req := httptest.NewRequest(http.MethodPost, "/v1/chat/completions", bytes.NewBufferString(body))
	rec := httptest.NewRecorder()

	handler(rec, req)

	// This should be a NavPlane error (502 Bad Gateway)
	if rec.Code != http.StatusBadGateway {
		t.Errorf("expected status 502, got %d", rec.Code)
	}

	assertJSONError(t, rec.Body.Bytes(), "failed to reach upstream provider", "server_error")
}

// --- Helper Types and Functions ---

type mockNetworkError struct {
	message string
}

func (e *mockNetworkError) Error() string {
	return e.message
}

func assertJSONError(t *testing.T, body []byte, expectedMessage, expectedType string) {
	t.Helper()

	var resp map[string]any
	if err := json.Unmarshal(body, &resp); err != nil {
		t.Fatalf("failed to parse error response: %v", err)
	}

	errObj, ok := resp["error"].(map[string]any)
	if !ok {
		t.Fatalf("expected error object, got %T", resp["error"])
	}

	if msg := errObj["message"].(string); msg != expectedMessage {
		t.Errorf("expected error message '%s', got '%s'", expectedMessage, msg)
	}

	if typ := errObj["type"].(string); typ != expectedType {
		t.Errorf("expected error type '%s', got '%s'", expectedType, typ)
	}
}
