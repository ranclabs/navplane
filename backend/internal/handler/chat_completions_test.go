package handler

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestChatCompletions_ValidRequest(t *testing.T) {
	// Test: POST with valid JSON and required fields returns 200
	body := `{
		"model": "gpt-4",
		"messages": [
			{"role": "user", "content": "Hello!"}
		]
	}`

	req := httptest.NewRequest(http.MethodPost, "/v1/chat/completions", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	ChatCompletions(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", rec.Code)
	}

	// Verify Content-Type header
	contentType := rec.Header().Get("Content-Type")
	if contentType != "application/json" {
		t.Errorf("expected Content-Type 'application/json', got '%s'", contentType)
	}

	// Verify response body
	var resp map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}
	if resp["status"] != "ok" {
		t.Errorf("expected status 'ok', got %v", resp["status"])
	}
}

func TestChatCompletions_ValidRequestWithOptionalFields(t *testing.T) {
	// Test: POST with all optional fields returns 200
	body := `{
		"model": "gpt-3.5-turbo",
		"messages": [
			{"role": "system", "content": "You are helpful."},
			{"role": "user", "content": "Hi"}
		],
		"stream": false,
		"temperature": 0.7,
		"max_tokens": 100,
		"top_p": 0.9
	}`

	req := httptest.NewRequest(http.MethodPost, "/v1/chat/completions", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	ChatCompletions(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", rec.Code)
	}
}

func TestChatCompletions_ValidRequestWithUnknownFields(t *testing.T) {
	// Test: POST with unknown fields still returns 200 (unknown fields preserved)
	body := `{
		"model": "gpt-4",
		"messages": [{"role": "user", "content": "Hello"}],
		"response_format": {"type": "json_object"},
		"seed": 42,
		"custom_field": "value"
	}`

	req := httptest.NewRequest(http.MethodPost, "/v1/chat/completions", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	ChatCompletions(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", rec.Code)
	}
}

func TestChatCompletions_InvalidJSON(t *testing.T) {
	// Test: POST with invalid JSON returns 400
	body := `{"model": "gpt-4", "messages": [`

	req := httptest.NewRequest(http.MethodPost, "/v1/chat/completions", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	ChatCompletions(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected status 400, got %d", rec.Code)
	}

	// Verify Content-Type header
	contentType := rec.Header().Get("Content-Type")
	if contentType != "application/json" {
		t.Errorf("expected Content-Type 'application/json', got '%s'", contentType)
	}

	// Verify error response structure
	var resp map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}
	errObj, ok := resp["error"].(map[string]any)
	if !ok {
		t.Fatalf("expected error object, got %T", resp["error"])
	}
	if errObj["type"] != "invalid_request_error" {
		t.Errorf("expected error type 'invalid_request_error', got %v", errObj["type"])
	}
	msg, ok := errObj["message"].(string)
	if !ok || !strings.Contains(msg, "invalid JSON") {
		t.Errorf("expected error message to contain 'invalid JSON', got %v", errObj["message"])
	}
}

func TestChatCompletions_MissingModel(t *testing.T) {
	// Test: POST missing model returns 400
	body := `{
		"messages": [{"role": "user", "content": "Hello"}]
	}`

	req := httptest.NewRequest(http.MethodPost, "/v1/chat/completions", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	ChatCompletions(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected status 400, got %d", rec.Code)
	}

	var resp map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}
	errObj := resp["error"].(map[string]any)
	msg := errObj["message"].(string)
	if !strings.Contains(msg, "model") {
		t.Errorf("expected error message to mention 'model', got %s", msg)
	}
}

func TestChatCompletions_EmptyModel(t *testing.T) {
	// Test: POST with empty model returns 400
	body := `{
		"model": "",
		"messages": [{"role": "user", "content": "Hello"}]
	}`

	req := httptest.NewRequest(http.MethodPost, "/v1/chat/completions", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	ChatCompletions(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected status 400, got %d", rec.Code)
	}

	var resp map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}
	errObj := resp["error"].(map[string]any)
	msg := errObj["message"].(string)
	if !strings.Contains(msg, "model") {
		t.Errorf("expected error message to mention 'model', got %s", msg)
	}
}

func TestChatCompletions_MissingMessages(t *testing.T) {
	// Test: POST missing messages returns 400
	body := `{
		"model": "gpt-4"
	}`

	req := httptest.NewRequest(http.MethodPost, "/v1/chat/completions", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	ChatCompletions(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected status 400, got %d", rec.Code)
	}

	var resp map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}
	errObj := resp["error"].(map[string]any)
	msg := errObj["message"].(string)
	if !strings.Contains(msg, "messages") {
		t.Errorf("expected error message to mention 'messages', got %s", msg)
	}
}

func TestChatCompletions_EmptyMessages(t *testing.T) {
	// Test: POST with empty messages array returns 400
	body := `{
		"model": "gpt-4",
		"messages": []
	}`

	req := httptest.NewRequest(http.MethodPost, "/v1/chat/completions", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	ChatCompletions(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected status 400, got %d", rec.Code)
	}

	var resp map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}
	errObj := resp["error"].(map[string]any)
	msg := errObj["message"].(string)
	if !strings.Contains(msg, "messages") {
		t.Errorf("expected error message to mention 'messages', got %s", msg)
	}
}

func TestChatCompletions_MessageMissingRole(t *testing.T) {
	// Test: POST with message missing role returns 400
	body := `{
		"model": "gpt-4",
		"messages": [{"content": "Hello"}]
	}`

	req := httptest.NewRequest(http.MethodPost, "/v1/chat/completions", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	ChatCompletions(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected status 400, got %d", rec.Code)
	}

	var resp map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}
	errObj := resp["error"].(map[string]any)
	msg := errObj["message"].(string)
	if !strings.Contains(msg, "role") {
		t.Errorf("expected error message to mention 'role', got %s", msg)
	}
}

func TestChatCompletions_MessageEmptyRole(t *testing.T) {
	// Test: POST with message having empty role returns 400
	body := `{
		"model": "gpt-4",
		"messages": [{"role": "", "content": "Hello"}]
	}`

	req := httptest.NewRequest(http.MethodPost, "/v1/chat/completions", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	ChatCompletions(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected status 400, got %d", rec.Code)
	}

	var resp map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}
	errObj := resp["error"].(map[string]any)
	msg := errObj["message"].(string)
	if !strings.Contains(msg, "role") {
		t.Errorf("expected error message to mention 'role', got %s", msg)
	}
}

func TestChatCompletions_MessageMissingContent(t *testing.T) {
	// Test: POST with message missing content returns 400
	body := `{
		"model": "gpt-4",
		"messages": [{"role": "user"}]
	}`

	req := httptest.NewRequest(http.MethodPost, "/v1/chat/completions", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	ChatCompletions(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected status 400, got %d", rec.Code)
	}

	var resp map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}
	errObj := resp["error"].(map[string]any)
	msg := errObj["message"].(string)
	if !strings.Contains(msg, "content") {
		t.Errorf("expected error message to mention 'content', got %s", msg)
	}
}

func TestChatCompletions_MessageEmptyContent(t *testing.T) {
	// Test: POST with message having empty content returns 400
	body := `{
		"model": "gpt-4",
		"messages": [{"role": "user", "content": ""}]
	}`

	req := httptest.NewRequest(http.MethodPost, "/v1/chat/completions", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	ChatCompletions(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected status 400, got %d", rec.Code)
	}

	var resp map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}
	errObj := resp["error"].(map[string]any)
	msg := errObj["message"].(string)
	if !strings.Contains(msg, "content") {
		t.Errorf("expected error message to mention 'content', got %s", msg)
	}
}

func TestChatCompletions_MethodNotAllowed_GET(t *testing.T) {
	// Test: GET returns 405
	req := httptest.NewRequest(http.MethodGet, "/v1/chat/completions", nil)
	rec := httptest.NewRecorder()

	ChatCompletions(rec, req)

	if rec.Code != http.StatusMethodNotAllowed {
		t.Errorf("expected status 405, got %d", rec.Code)
	}

	// Verify Content-Type header
	contentType := rec.Header().Get("Content-Type")
	if contentType != "application/json" {
		t.Errorf("expected Content-Type 'application/json', got '%s'", contentType)
	}

	var resp map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}
	errObj, ok := resp["error"].(map[string]any)
	if !ok {
		t.Fatalf("expected error object, got %T", resp["error"])
	}
	if errObj["type"] != "invalid_request_error" {
		t.Errorf("expected error type 'invalid_request_error', got %v", errObj["type"])
	}
}

func TestChatCompletions_MethodNotAllowed_PUT(t *testing.T) {
	// Test: PUT returns 405
	body := `{"model": "gpt-4", "messages": [{"role": "user", "content": "Hello"}]}`
	req := httptest.NewRequest(http.MethodPut, "/v1/chat/completions", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	ChatCompletions(rec, req)

	if rec.Code != http.StatusMethodNotAllowed {
		t.Errorf("expected status 405, got %d", rec.Code)
	}
}

func TestChatCompletions_MethodNotAllowed_DELETE(t *testing.T) {
	// Test: DELETE returns 405
	req := httptest.NewRequest(http.MethodDelete, "/v1/chat/completions", nil)
	rec := httptest.NewRecorder()

	ChatCompletions(rec, req)

	if rec.Code != http.StatusMethodNotAllowed {
		t.Errorf("expected status 405, got %d", rec.Code)
	}
}

func TestChatCompletions_MethodNotAllowed_PATCH(t *testing.T) {
	// Test: PATCH returns 405
	req := httptest.NewRequest(http.MethodPatch, "/v1/chat/completions", nil)
	rec := httptest.NewRecorder()

	ChatCompletions(rec, req)

	if rec.Code != http.StatusMethodNotAllowed {
		t.Errorf("expected status 405, got %d", rec.Code)
	}
}

func TestChatCompletions_MultipleMessages(t *testing.T) {
	// Test: POST with multiple messages returns 200
	body := `{
		"model": "gpt-4",
		"messages": [
			{"role": "system", "content": "You are a helpful assistant."},
			{"role": "user", "content": "Hello"},
			{"role": "assistant", "content": "Hi there!"},
			{"role": "user", "content": "How are you?"}
		]
	}`

	req := httptest.NewRequest(http.MethodPost, "/v1/chat/completions", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	ChatCompletions(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", rec.Code)
	}
}

func TestChatCompletions_SecondMessageMissingRole(t *testing.T) {
	// Test: Validation catches errors in non-first messages
	body := `{
		"model": "gpt-4",
		"messages": [
			{"role": "user", "content": "First message"},
			{"content": "Second message without role"}
		]
	}`

	req := httptest.NewRequest(http.MethodPost, "/v1/chat/completions", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	ChatCompletions(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected status 400, got %d", rec.Code)
	}

	var resp map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}
	errObj := resp["error"].(map[string]any)
	msg := errObj["message"].(string)
	if !strings.Contains(msg, "index 1") {
		t.Errorf("expected error message to mention 'index 1', got %s", msg)
	}
}

func TestChatCompletions_EmptyBody(t *testing.T) {
	// Test: Empty body returns 400
	req := httptest.NewRequest(http.MethodPost, "/v1/chat/completions", bytes.NewBufferString(""))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	ChatCompletions(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected status 400, got %d", rec.Code)
	}
}
