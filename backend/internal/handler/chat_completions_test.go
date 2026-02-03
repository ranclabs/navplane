package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"navplane/internal/config"
)

// roundTripperFunc allows using a function as an http.RoundTripper for testing.
type roundTripperFunc func(*http.Request) (*http.Response, error)

func (f roundTripperFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}

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

func mockHTTPClient(handler func(req *http.Request) (*http.Response, error)) *http.Client {
	return &http.Client{Transport: roundTripperFunc(handler)}
}

// --- Method Not Allowed Tests ---

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

// --- Non-Streaming Passthrough Tests ---

func TestChatCompletions_PassthroughSuccess(t *testing.T) {
	cfg := testConfig()

	upstreamResponse := map[string]any{
		"id":      "chatcmpl-123",
		"object":  "chat.completion",
		"created": 1677652288,
		"model":   "gpt-4",
		"choices": []map[string]any{
			{"index": 0, "message": map[string]any{"role": "assistant", "content": "Hello!"}, "finish_reason": "stop"},
		},
	}
	upstreamBody, _ := json.Marshal(upstreamResponse)

	client := mockHTTPClient(func(req *http.Request) (*http.Response, error) {
		return &http.Response{
			StatusCode: http.StatusOK,
			Header:     http.Header{"Content-Type": []string{"application/json"}},
			Body:       io.NopCloser(bytes.NewReader(upstreamBody)),
		}, nil
	})

	handler := NewChatCompletionsHandlerWithClient(cfg, client)

	body := `{"model": "gpt-4", "messages": [{"role": "user", "content": "Hello!"}]}`
	req := httptest.NewRequest(http.MethodPost, "/v1/chat/completions", bytes.NewBufferString(body))
	rec := httptest.NewRecorder()

	handler(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", rec.Code)
	}

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

	upstreamError := map[string]any{
		"error": map[string]any{
			"message": "The model `gpt-5` does not exist",
			"type":    "invalid_request_error",
			"code":    "model_not_found",
		},
	}
	upstreamBody, _ := json.Marshal(upstreamError)

	client := mockHTTPClient(func(req *http.Request) (*http.Response, error) {
		return &http.Response{
			StatusCode: http.StatusNotFound,
			Header:     http.Header{"Content-Type": []string{"application/json"}},
			Body:       io.NopCloser(bytes.NewReader(upstreamBody)),
		}, nil
	})

	handler := NewChatCompletionsHandlerWithClient(cfg, client)

	body := `{"model": "gpt-5", "messages": [{"role": "user", "content": "Hello!"}]}`
	req := httptest.NewRequest(http.MethodPost, "/v1/chat/completions", bytes.NewBufferString(body))
	rec := httptest.NewRecorder()

	handler(rec, req)

	// Upstream error passed through
	if rec.Code != http.StatusNotFound {
		t.Errorf("expected status 404, got %d", rec.Code)
	}
}

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
	req.Header.Set("Authorization", "Bearer client-secret-token-should-not-forward")
	rec := httptest.NewRecorder()

	handler(rec, req)

	expectedAuth := "Bearer " + cfg.Provider.APIKey
	if capturedAuthHeader != expectedAuth {
		t.Errorf("expected upstream to receive '%s', got '%s'", expectedAuth, capturedAuthHeader)
	}
}

func TestChatCompletions_RequestBodyPassthrough(t *testing.T) {
	cfg := testConfig()

	var capturedBody []byte
	client := mockHTTPClient(func(req *http.Request) (*http.Response, error) {
		capturedBody, _ = io.ReadAll(req.Body)
		return &http.Response{
			StatusCode: http.StatusOK,
			Header:     http.Header{"Content-Type": []string{"application/json"}},
			Body:       io.NopCloser(bytes.NewReader([]byte(`{"id":"test"}`))),
		}, nil
	})

	handler := NewChatCompletionsHandlerWithClient(cfg, client)

	originalBody := `{"model":"gpt-4","messages":[{"role":"user","content":"Hello"}],"custom_field":"passthrough"}`
	req := httptest.NewRequest(http.MethodPost, "/v1/chat/completions", bytes.NewBufferString(originalBody))
	rec := httptest.NewRecorder()

	handler(rec, req)

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
		{"invalid json", `{invalid}`, false},
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

func TestChatCompletions_StreamingResponse(t *testing.T) {
	cfg := testConfig()

	streamingResponse := "data: {\"id\":\"chatcmpl-123\"}\n\ndata: [DONE]\n\n"

	client := mockHTTPClient(func(req *http.Request) (*http.Response, error) {
		return &http.Response{
			StatusCode: http.StatusOK,
			Header:     http.Header{"Content-Type": []string{"text/event-stream"}},
			Body:       io.NopCloser(strings.NewReader(streamingResponse)),
		}, nil
	})

	handler := NewChatCompletionsHandlerWithClient(cfg, client)

	body := `{"model": "gpt-4", "messages": [{"role": "user", "content": "Hi"}], "stream": true}`
	req := httptest.NewRequest(http.MethodPost, "/v1/chat/completions", bytes.NewBufferString(body))
	rec := httptest.NewRecorder()

	handler(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", rec.Code)
	}

	if ct := rec.Header().Get("Content-Type"); ct != "text/event-stream" {
		t.Errorf("expected Content-Type 'text/event-stream', got '%s'", ct)
	}

	if rec.Body.String() != streamingResponse {
		t.Errorf("streaming response was modified!\nexpected: %q\ngot: %q", streamingResponse, rec.Body.String())
	}
}

func TestChatCompletions_StreamingUpstreamError(t *testing.T) {
	cfg := testConfig()

	upstreamError := map[string]any{
		"error": map[string]any{"message": "Rate limit exceeded", "type": "rate_limit_error"},
	}
	upstreamBody, _ := json.Marshal(upstreamError)

	client := mockHTTPClient(func(req *http.Request) (*http.Response, error) {
		return &http.Response{
			StatusCode: http.StatusTooManyRequests,
			Header:     http.Header{"Content-Type": []string{"application/json"}},
			Body:       io.NopCloser(bytes.NewReader(upstreamBody)),
		}, nil
	})

	handler := NewChatCompletionsHandlerWithClient(cfg, client)

	body := `{"model": "gpt-4", "messages": [{"role": "user", "content": "Hi"}], "stream": true}`
	req := httptest.NewRequest(http.MethodPost, "/v1/chat/completions", bytes.NewBufferString(body))
	rec := httptest.NewRecorder()

	handler(rec, req)

	// Non-200 upstream error passed through (not as SSE)
	if rec.Code != http.StatusTooManyRequests {
		t.Errorf("expected status 429, got %d", rec.Code)
	}

	// Should NOT be SSE content type for errors
	if ct := rec.Header().Get("Content-Type"); ct == "text/event-stream" {
		t.Error("error response should not use text/event-stream")
	}
}

// --- Error Handling Tests ---

func TestChatCompletions_RequestBodyTooLarge(t *testing.T) {
	cfg := testConfig()
	client := mockHTTPClient(func(req *http.Request) (*http.Response, error) {
		t.Fatal("upstream should not be called for oversized request")
		return nil, nil
	})

	handler := NewChatCompletionsHandlerWithClient(cfg, client)

	largeBody := strings.Repeat("x", 11*1024*1024)
	req := httptest.NewRequest(http.MethodPost, "/v1/chat/completions", strings.NewReader(largeBody))
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

	if rec.Code != http.StatusBadGateway {
		t.Errorf("expected status 502, got %d", rec.Code)
	}
	assertJSONError(t, rec.Body.Bytes(), "failed to reach upstream provider", "server_error")
}

// --- Client Disconnection Tests ---

func TestChatCompletions_StreamingClientDisconnect(t *testing.T) {
	cfg := testConfig()

	upstreamCancelled := make(chan struct{})
	client := mockHTTPClient(func(req *http.Request) (*http.Response, error) {
		<-req.Context().Done()
		close(upstreamCancelled)
		return nil, req.Context().Err()
	})

	handler := NewChatCompletionsHandlerWithClient(cfg, client)

	ctx, cancel := context.WithCancel(context.Background())
	body := `{"stream": true, "model": "gpt-4", "messages": [{"role": "user", "content": "Hi"}]}`
	req := httptest.NewRequest(http.MethodPost, "/v1/chat/completions", bytes.NewBufferString(body))
	req = req.WithContext(ctx)
	rec := httptest.NewRecorder()

	go func() {
		time.Sleep(10 * time.Millisecond)
		cancel() // Simulate client disconnect
	}()

	handler(rec, req)

	select {
	case <-upstreamCancelled:
		// Success: upstream request was cancelled
	case <-time.After(time.Second):
		t.Error("upstream request was not cancelled when client disconnected")
	}
}

// --- URL Normalization Tests ---

func TestChatCompletions_URLNormalization(t *testing.T) {
	tests := []struct {
		name        string
		baseURL     string
		expectedURL string
	}{
		{"no trailing slash", "https://api.openai.com", "https://api.openai.com/v1/chat/completions"},
		{"trailing slash", "https://api.openai.com/", "https://api.openai.com/v1/chat/completions"},
		{"includes v1", "https://api.openai.com/v1", "https://api.openai.com/v1/chat/completions"},
		{"includes v1 with slash", "https://api.openai.com/v1/", "https://api.openai.com/v1/chat/completions"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var capturedURL string
			client := mockHTTPClient(func(req *http.Request) (*http.Response, error) {
				capturedURL = req.URL.String()
				return &http.Response{
					StatusCode: http.StatusOK,
					Header:     http.Header{"Content-Type": []string{"application/json"}},
					Body:       io.NopCloser(bytes.NewReader([]byte(`{"id":"test"}`))),
				}, nil
			})

			cfg := &config.Config{
				Provider: config.ProviderConfig{
					BaseURL: tt.baseURL,
					APIKey:  "test-key",
				},
			}

			handler := NewChatCompletionsHandlerWithClient(cfg, client)
			body := `{"model": "gpt-4", "messages": [{"role": "user", "content": "Hi"}]}`
			req := httptest.NewRequest(http.MethodPost, "/v1/chat/completions", bytes.NewBufferString(body))
			rec := httptest.NewRecorder()

			handler(rec, req)

			if capturedURL != tt.expectedURL {
				t.Errorf("expected URL %s, got %s", tt.expectedURL, capturedURL)
			}
		})
	}
}

// --- Helpers ---

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
