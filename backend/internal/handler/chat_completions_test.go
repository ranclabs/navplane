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

	"navplane/internal/middleware"
	"navplane/internal/org"
	"navplane/internal/provider"
	"navplane/internal/providerkey"

	"github.com/google/uuid"
)

// roundTripperFunc allows using a function as an http.RoundTripper for testing.
type roundTripperFunc func(*http.Request) (*http.Response, error)

func (f roundTripperFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}

// mockProviderKeyManager is a simple mock for testing.
type mockProviderKeyManager struct {
	keys map[string]string // provider -> decrypted key
}

func (m *mockProviderKeyManager) GetDecryptedKey(ctx context.Context, orgID uuid.UUID, providerName string) (string, error) {
	if key, ok := m.keys[providerName]; ok {
		return key, nil
	}
	return "", providerkey.ErrNotFound
}

func (m *mockProviderKeyManager) ListByOrg(ctx context.Context, orgID uuid.UUID) ([]*providerkey.ProviderKey, error) {
	return nil, nil
}

func (m *mockProviderKeyManager) Create(ctx context.Context, input providerkey.CreateInput) (*providerkey.ProviderKey, error) {
	return nil, nil
}

func (m *mockProviderKeyManager) Delete(ctx context.Context, id uuid.UUID) error {
	return nil
}

func testDeps() *ChatCompletionsDeps {
	return &ChatCompletionsDeps{
		ProviderRegistry: provider.NewRegistry(),
		ProviderKeyManager: &mockProviderKeyManagerWrapper{
			keys: map[string]string{
				"openai":    "sk-test-key-for-testing-only",
				"anthropic": "sk-ant-test-key",
			},
		},
	}
}

// mockProviderKeyManagerWrapper wraps mockProviderKeyManager to implement the interface.
type mockProviderKeyManagerWrapper struct {
	keys map[string]string
}

func (m *mockProviderKeyManagerWrapper) GetDecryptedKey(ctx context.Context, orgID uuid.UUID, providerName string) (string, error) {
	if key, ok := m.keys[providerName]; ok {
		return key, nil
	}
	return "", providerkey.ErrNotFound
}

func mockHTTPClient(handler func(req *http.Request) (*http.Response, error)) *http.Client {
	return &http.Client{Transport: roundTripperFunc(handler)}
}

// withOrgContext adds a mock org to the request context.
func withOrgContext(req *http.Request) *http.Request {
	testOrg := &org.Org{
		ID:      uuid.New(),
		Name:    "Test Org",
		Enabled: true,
	}
	ctx := context.WithValue(req.Context(), middleware.OrgContextKey, testOrg)
	return req.WithContext(ctx)
}

// newTestHandler creates a handler for testing with mocked dependencies.
func newTestHandler(client *http.Client, keys map[string]string) http.HandlerFunc {
	deps := &ChatCompletionsDeps{
		ProviderRegistry: provider.NewRegistry(),
		ProviderKeyManager: &mockProviderKeyManagerWrapper{
			keys: keys,
		},
	}
	return newChatHandler(deps, client).ServeHTTP
}

// --- Method Not Allowed Tests ---

func TestChatCompletions_MethodNotAllowed_GET(t *testing.T) {
	client := mockHTTPClient(func(req *http.Request) (*http.Response, error) {
		t.Fatal("upstream should not be called for GET request")
		return nil, nil
	})

	handler := newTestHandler(client, map[string]string{"openai": "test-key"})
	req := httptest.NewRequest(http.MethodGet, "/v1/chat/completions", nil)
	req = withOrgContext(req)
	rec := httptest.NewRecorder()

	handler(rec, req)

	if rec.Code != http.StatusMethodNotAllowed {
		t.Errorf("expected status 405, got %d", rec.Code)
	}
	assertJSONError(t, rec.Body.Bytes(), "method not allowed", "invalid_request_error")
}

// --- Non-Streaming Passthrough Tests ---

func TestChatCompletions_PassthroughSuccess(t *testing.T) {
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

	handler := newTestHandler(client, map[string]string{"openai": "sk-test-key"})

	body := `{"model": "gpt-4", "messages": [{"role": "user", "content": "Hello!"}]}`
	req := httptest.NewRequest(http.MethodPost, "/v1/chat/completions", bytes.NewBufferString(body))
	req = withOrgContext(req)
	rec := httptest.NewRecorder()

	handler(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d: %s", rec.Code, rec.Body.String())
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

	handler := newTestHandler(client, map[string]string{"openai": "sk-test-key"})

	body := `{"model": "gpt-4", "messages": [{"role": "user", "content": "Hello!"}]}`
	req := httptest.NewRequest(http.MethodPost, "/v1/chat/completions", bytes.NewBufferString(body))
	req = withOrgContext(req)
	rec := httptest.NewRecorder()

	handler(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Errorf("expected status 404, got %d", rec.Code)
	}
}

func TestChatCompletions_ClientAuthNeverForwarded(t *testing.T) {
	var capturedAuthHeader string
	client := mockHTTPClient(func(req *http.Request) (*http.Response, error) {
		capturedAuthHeader = req.Header.Get("Authorization")
		return &http.Response{
			StatusCode: http.StatusOK,
			Header:     http.Header{"Content-Type": []string{"application/json"}},
			Body:       io.NopCloser(bytes.NewReader([]byte(`{"id":"test"}`))),
		}, nil
	})

	expectedKey := "sk-test-key-for-org"
	handler := newTestHandler(client, map[string]string{"openai": expectedKey})

	body := `{"model": "gpt-4", "messages": [{"role": "user", "content": "Hello"}]}`
	req := httptest.NewRequest(http.MethodPost, "/v1/chat/completions", bytes.NewBufferString(body))
	req.Header.Set("Authorization", "Bearer client-secret-token-should-not-forward")
	req = withOrgContext(req)
	rec := httptest.NewRecorder()

	handler(rec, req)

	expectedAuth := "Bearer " + expectedKey
	if capturedAuthHeader != expectedAuth {
		t.Errorf("expected upstream to receive '%s', got '%s'", expectedAuth, capturedAuthHeader)
	}
}

func TestChatCompletions_NoProviderKey(t *testing.T) {
	client := mockHTTPClient(func(req *http.Request) (*http.Response, error) {
		t.Fatal("upstream should not be called when no provider key")
		return nil, nil
	})

	// No keys configured
	handler := newTestHandler(client, map[string]string{})

	body := `{"model": "gpt-4", "messages": [{"role": "user", "content": "Hello"}]}`
	req := httptest.NewRequest(http.MethodPost, "/v1/chat/completions", bytes.NewBufferString(body))
	req = withOrgContext(req)
	rec := httptest.NewRecorder()

	handler(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected status 400, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestChatCompletions_RequestBodyPassthrough(t *testing.T) {
	var capturedBody []byte
	client := mockHTTPClient(func(req *http.Request) (*http.Response, error) {
		capturedBody, _ = io.ReadAll(req.Body)
		return &http.Response{
			StatusCode: http.StatusOK,
			Header:     http.Header{"Content-Type": []string{"application/json"}},
			Body:       io.NopCloser(bytes.NewReader([]byte(`{"id":"test"}`))),
		}, nil
	})

	handler := newTestHandler(client, map[string]string{"openai": "sk-test-key"})

	originalBody := `{"model":"gpt-4","messages":[{"role":"user","content":"Hello"}],"custom_field":"passthrough"}`
	req := httptest.NewRequest(http.MethodPost, "/v1/chat/completions", bytes.NewBufferString(originalBody))
	req = withOrgContext(req)
	rec := httptest.NewRecorder()

	handler(rec, req)

	if string(capturedBody) != originalBody {
		t.Errorf("request body was modified!\nexpected: %s\ngot: %s", originalBody, string(capturedBody))
	}
}

// --- Streaming Tests ---

func TestChatCompletions_StreamingResponse(t *testing.T) {
	streamingResponse := "data: {\"id\":\"chatcmpl-123\"}\n\ndata: [DONE]\n\n"

	client := mockHTTPClient(func(req *http.Request) (*http.Response, error) {
		return &http.Response{
			StatusCode: http.StatusOK,
			Header:     http.Header{"Content-Type": []string{"text/event-stream"}},
			Body:       io.NopCloser(strings.NewReader(streamingResponse)),
		}, nil
	})

	handler := newTestHandler(client, map[string]string{"openai": "sk-test-key"})

	body := `{"model": "gpt-4", "messages": [{"role": "user", "content": "Hi"}], "stream": true}`
	req := httptest.NewRequest(http.MethodPost, "/v1/chat/completions", bytes.NewBufferString(body))
	req = withOrgContext(req)
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

	handler := newTestHandler(client, map[string]string{"openai": "sk-test-key"})

	body := `{"model": "gpt-4", "messages": [{"role": "user", "content": "Hi"}], "stream": true}`
	req := httptest.NewRequest(http.MethodPost, "/v1/chat/completions", bytes.NewBufferString(body))
	req = withOrgContext(req)
	rec := httptest.NewRecorder()

	handler(rec, req)

	if rec.Code != http.StatusTooManyRequests {
		t.Errorf("expected status 429, got %d", rec.Code)
	}

	if ct := rec.Header().Get("Content-Type"); ct == "text/event-stream" {
		t.Error("error response should not use text/event-stream")
	}
}

// --- Error Handling Tests ---

func TestChatCompletions_RequestBodyTooLarge(t *testing.T) {
	client := mockHTTPClient(func(req *http.Request) (*http.Response, error) {
		t.Fatal("upstream should not be called for oversized request")
		return nil, nil
	})

	handler := newTestHandler(client, map[string]string{"openai": "sk-test-key"})

	largeBody := `{"model":"gpt-4","messages":[{"role":"user","content":"` + strings.Repeat("x", 11*1024*1024) + `"}]}`
	req := httptest.NewRequest(http.MethodPost, "/v1/chat/completions", strings.NewReader(largeBody))
	req = withOrgContext(req)
	rec := httptest.NewRecorder()

	handler(rec, req)

	if rec.Code != http.StatusRequestEntityTooLarge {
		t.Errorf("expected status 413, got %d", rec.Code)
	}
	assertJSONError(t, rec.Body.Bytes(), "request body too large", "invalid_request_error")
}

func TestChatCompletions_UpstreamUnreachable(t *testing.T) {
	client := mockHTTPClient(func(req *http.Request) (*http.Response, error) {
		return nil, &mockNetworkError{message: "connection refused"}
	})

	handler := newTestHandler(client, map[string]string{"openai": "sk-test-key"})

	body := `{"model": "gpt-4", "messages": [{"role": "user", "content": "Hello"}]}`
	req := httptest.NewRequest(http.MethodPost, "/v1/chat/completions", bytes.NewBufferString(body))
	req = withOrgContext(req)
	rec := httptest.NewRecorder()

	handler(rec, req)

	if rec.Code != http.StatusBadGateway {
		t.Errorf("expected status 502, got %d", rec.Code)
	}
	assertJSONError(t, rec.Body.Bytes(), "failed to reach upstream provider", "server_error")
}

// --- Client Disconnection Tests ---

func TestChatCompletions_StreamingClientDisconnect(t *testing.T) {
	upstreamCancelled := make(chan struct{})
	client := mockHTTPClient(func(req *http.Request) (*http.Response, error) {
		<-req.Context().Done()
		close(upstreamCancelled)
		return nil, req.Context().Err()
	})

	handler := newTestHandler(client, map[string]string{"openai": "sk-test-key"})

	ctx, cancel := context.WithCancel(context.Background())
	body := `{"stream": true, "model": "gpt-4", "messages": [{"role": "user", "content": "Hi"}]}`
	req := httptest.NewRequest(http.MethodPost, "/v1/chat/completions", bytes.NewBufferString(body))
	req = req.WithContext(ctx)
	req = withOrgContext(req)
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

// --- Provider Routing Tests ---

func TestChatCompletions_RoutesToCorrectProvider(t *testing.T) {
	tests := []struct {
		model            string
		expectedProvider string
		expectedAuthKey  string
	}{
		{"gpt-4", "openai", "Bearer sk-openai-key"},
		{"gpt-4o", "openai", "Bearer sk-openai-key"},
		{"claude-3-5-sonnet-20241022", "anthropic", "sk-anthropic-key"},
	}

	for _, tt := range tests {
		t.Run(tt.model, func(t *testing.T) {
			var capturedAuthHeader string
			var capturedURL string
			client := mockHTTPClient(func(req *http.Request) (*http.Response, error) {
				capturedAuthHeader = req.Header.Get("Authorization")
				if capturedAuthHeader == "" {
					capturedAuthHeader = req.Header.Get("x-api-key")
				}
				capturedURL = req.URL.String()
				return &http.Response{
					StatusCode: http.StatusOK,
					Header:     http.Header{"Content-Type": []string{"application/json"}},
					Body:       io.NopCloser(bytes.NewReader([]byte(`{"id":"test"}`))),
				}, nil
			})

			handler := newTestHandler(client, map[string]string{
				"openai":    "sk-openai-key",
				"anthropic": "sk-anthropic-key",
			})

			body := `{"model": "` + tt.model + `", "messages": [{"role": "user", "content": "Hi"}]}`
			req := httptest.NewRequest(http.MethodPost, "/v1/chat/completions", bytes.NewBufferString(body))
			req = withOrgContext(req)
			rec := httptest.NewRecorder()

			handler(rec, req)

			if capturedAuthHeader != tt.expectedAuthKey {
				t.Errorf("expected auth '%s', got '%s'", tt.expectedAuthKey, capturedAuthHeader)
			}

			if tt.expectedProvider == "openai" && !strings.Contains(capturedURL, "api.openai.com") {
				t.Errorf("expected OpenAI URL, got %s", capturedURL)
			}
			if tt.expectedProvider == "anthropic" && !strings.Contains(capturedURL, "api.anthropic.com") {
				t.Errorf("expected Anthropic URL, got %s", capturedURL)
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
		t.Fatalf("failed to parse error response: %v, body: %s", err, string(body))
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
