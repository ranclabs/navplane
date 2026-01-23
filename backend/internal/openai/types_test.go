package openai

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestChatCompletionsRequest_MinimalFields(t *testing.T) {
	// Test: Minimal request with only model and messages
	input := `{
		"model": "gpt-4",
		"messages": [
			{"role": "system", "content": "You are a helpful assistant."},
			{"role": "user", "content": "Hello!"}
		]
	}`

	var req ChatCompletionsRequest
	err := json.Unmarshal([]byte(input), &req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Verify model
	if req.Model != "gpt-4" {
		t.Errorf("expected model 'gpt-4', got '%s'", req.Model)
	}

	// Verify messages
	if len(req.Messages) != 2 {
		t.Fatalf("expected 2 messages, got %d", len(req.Messages))
	}
	if req.Messages[0].Role != "system" {
		t.Errorf("expected first message role 'system', got '%s'", req.Messages[0].Role)
	}
	if req.Messages[0].Content != "You are a helpful assistant." {
		t.Errorf("unexpected first message content: %s", req.Messages[0].Content)
	}
	if req.Messages[1].Role != "user" {
		t.Errorf("expected second message role 'user', got '%s'", req.Messages[1].Role)
	}
	if req.Messages[1].Content != "Hello!" {
		t.Errorf("unexpected second message content: %s", req.Messages[1].Content)
	}

	// Verify optional fields are nil (unset)
	if req.Stream != nil {
		t.Errorf("expected stream to be nil, got %v", *req.Stream)
	}
	if req.Temperature != nil {
		t.Errorf("expected temperature to be nil, got %v", *req.Temperature)
	}
	if req.MaxTokens != nil {
		t.Errorf("expected max_tokens to be nil, got %v", *req.MaxTokens)
	}
	if req.TopP != nil {
		t.Errorf("expected top_p to be nil, got %v", *req.TopP)
	}

	// Verify no extra fields
	if req.Extra != nil {
		t.Errorf("expected Extra to be nil, got %v", req.Extra)
	}
}

func TestChatCompletionsRequest_AllKnownFields(t *testing.T) {
	// Test: Request with stream, temperature, max_tokens, top_p
	input := `{
		"model": "gpt-3.5-turbo",
		"messages": [{"role": "user", "content": "Test"}],
		"stream": true,
		"temperature": 0.7,
		"max_tokens": 1000,
		"top_p": 0.9
	}`

	var req ChatCompletionsRequest
	err := json.Unmarshal([]byte(input), &req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Verify model
	if req.Model != "gpt-3.5-turbo" {
		t.Errorf("expected model 'gpt-3.5-turbo', got '%s'", req.Model)
	}

	// Verify stream
	if req.Stream == nil {
		t.Fatal("expected stream to be set")
	}
	if *req.Stream != true {
		t.Errorf("expected stream true, got %v", *req.Stream)
	}

	// Verify temperature
	if req.Temperature == nil {
		t.Fatal("expected temperature to be set")
	}
	if *req.Temperature != 0.7 {
		t.Errorf("expected temperature 0.7, got %v", *req.Temperature)
	}

	// Verify max_tokens
	if req.MaxTokens == nil {
		t.Fatal("expected max_tokens to be set")
	}
	if *req.MaxTokens != 1000 {
		t.Errorf("expected max_tokens 1000, got %v", *req.MaxTokens)
	}

	// Verify top_p
	if req.TopP == nil {
		t.Fatal("expected top_p to be set")
	}
	if *req.TopP != 0.9 {
		t.Errorf("expected top_p 0.9, got %v", *req.TopP)
	}
}

func TestChatCompletionsRequest_ExplicitFalseAndZero(t *testing.T) {
	// Test: Verify pointer types distinguish explicit zero/false from unset
	input := `{
		"model": "gpt-4",
		"messages": [{"role": "user", "content": "Test"}],
		"stream": false,
		"temperature": 0,
		"max_tokens": 0,
		"top_p": 0
	}`

	var req ChatCompletionsRequest
	err := json.Unmarshal([]byte(input), &req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// All optional fields should be set (non-nil) even when zero/false
	if req.Stream == nil {
		t.Fatal("expected stream to be set (explicit false)")
	}
	if *req.Stream != false {
		t.Errorf("expected stream false, got %v", *req.Stream)
	}

	if req.Temperature == nil {
		t.Fatal("expected temperature to be set (explicit 0)")
	}
	if *req.Temperature != 0 {
		t.Errorf("expected temperature 0, got %v", *req.Temperature)
	}

	if req.MaxTokens == nil {
		t.Fatal("expected max_tokens to be set (explicit 0)")
	}
	if *req.MaxTokens != 0 {
		t.Errorf("expected max_tokens 0, got %v", *req.MaxTokens)
	}

	if req.TopP == nil {
		t.Fatal("expected top_p to be set (explicit 0)")
	}
	if *req.TopP != 0 {
		t.Errorf("expected top_p 0, got %v", *req.TopP)
	}
}

func TestChatCompletionsRequest_UnknownFields(t *testing.T) {
	// Test: Request containing unknown fields must not error and must preserve in Extra
	input := `{
		"model": "gpt-4",
		"messages": [{"role": "user", "content": "Hello"}],
		"temperature": 0.5,
		"metadata": {"user_id": "12345", "session": "abc"},
		"response_format": {"type": "json_object"},
		"foo": "bar",
		"seed": 42,
		"tools": [{"type": "function", "function": {"name": "get_weather"}}]
	}`

	var req ChatCompletionsRequest
	err := json.Unmarshal([]byte(input), &req)
	if err != nil {
		t.Fatalf("unexpected error unmarshalling with unknown fields: %v", err)
	}

	// Verify known fields are still parsed correctly
	if req.Model != "gpt-4" {
		t.Errorf("expected model 'gpt-4', got '%s'", req.Model)
	}
	if len(req.Messages) != 1 {
		t.Fatalf("expected 1 message, got %d", len(req.Messages))
	}
	if req.Temperature == nil || *req.Temperature != 0.5 {
		t.Errorf("expected temperature 0.5, got %v", req.Temperature)
	}

	// Verify Extra contains all unknown fields
	if req.Extra == nil {
		t.Fatal("expected Extra to be non-nil")
	}

	// Check each unknown field
	expectedUnknownFields := []string{"metadata", "response_format", "foo", "seed", "tools"}
	for _, field := range expectedUnknownFields {
		if _, exists := req.Extra[field]; !exists {
			t.Errorf("expected Extra to contain '%s'", field)
		}
	}

	// Verify specific unknown field values
	if foo, ok := req.Extra["foo"].(string); !ok || foo != "bar" {
		t.Errorf("expected Extra['foo'] = 'bar', got %v", req.Extra["foo"])
	}

	if seed, ok := req.Extra["seed"].(float64); !ok || seed != 42 {
		t.Errorf("expected Extra['seed'] = 42, got %v", req.Extra["seed"])
	}

	// Verify nested objects are preserved
	metadata, ok := req.Extra["metadata"].(map[string]any)
	if !ok {
		t.Fatalf("expected Extra['metadata'] to be a map, got %T", req.Extra["metadata"])
	}
	if metadata["user_id"] != "12345" {
		t.Errorf("expected metadata.user_id = '12345', got %v", metadata["user_id"])
	}

	// Verify response_format is preserved
	respFormat, ok := req.Extra["response_format"].(map[string]any)
	if !ok {
		t.Fatalf("expected Extra['response_format'] to be a map, got %T", req.Extra["response_format"])
	}
	if respFormat["type"] != "json_object" {
		t.Errorf("expected response_format.type = 'json_object', got %v", respFormat["type"])
	}

	// Verify tools array is preserved
	tools, ok := req.Extra["tools"].([]any)
	if !ok {
		t.Fatalf("expected Extra['tools'] to be an array, got %T", req.Extra["tools"])
	}
	if len(tools) != 1 {
		t.Errorf("expected 1 tool, got %d", len(tools))
	}
}

func TestChatCompletionsRequest_MarshalPreservesUnknownFields(t *testing.T) {
	// Test: Marshalling should include Extra fields for passthrough
	input := `{
		"model": "gpt-4",
		"messages": [{"role": "user", "content": "Hello"}],
		"temperature": 0.8,
		"custom_field": "custom_value",
		"nested": {"key": "value"}
	}`

	var req ChatCompletionsRequest
	err := json.Unmarshal([]byte(input), &req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Marshal back to JSON
	output, err := json.Marshal(req)
	if err != nil {
		t.Fatalf("unexpected error marshalling: %v", err)
	}

	// Unmarshal into a generic map to verify all fields are present
	var result map[string]any
	err = json.Unmarshal(output, &result)
	if err != nil {
		t.Fatalf("unexpected error unmarshalling result: %v", err)
	}

	// Verify known fields
	if result["model"] != "gpt-4" {
		t.Errorf("expected model 'gpt-4', got %v", result["model"])
	}
	if result["temperature"] != 0.8 {
		t.Errorf("expected temperature 0.8, got %v", result["temperature"])
	}

	// Verify unknown fields are preserved in output
	if result["custom_field"] != "custom_value" {
		t.Errorf("expected custom_field 'custom_value', got %v", result["custom_field"])
	}
	nested, ok := result["nested"].(map[string]any)
	if !ok {
		t.Fatalf("expected nested to be a map, got %T", result["nested"])
	}
	if nested["key"] != "value" {
		t.Errorf("expected nested.key 'value', got %v", nested["key"])
	}
}

func TestChatCompletionsResponse_Decode(t *testing.T) {
	// Test: Response decode with typical OpenAI response JSON
	input := `{
		"id": "chatcmpl-abc123",
		"object": "chat.completion",
		"created": 1677858242,
		"model": "gpt-4-0613",
		"choices": [
			{
				"index": 0,
				"message": {
					"role": "assistant",
					"content": "Hello! How can I assist you today?"
				},
				"finish_reason": "stop"
			}
		],
		"usage": {
			"prompt_tokens": 13,
			"completion_tokens": 9,
			"total_tokens": 22
		}
	}`

	var resp ChatCompletionsResponse
	err := json.Unmarshal([]byte(input), &resp)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Verify top-level fields
	if resp.ID != "chatcmpl-abc123" {
		t.Errorf("expected id 'chatcmpl-abc123', got '%s'", resp.ID)
	}
	if resp.Object != "chat.completion" {
		t.Errorf("expected object 'chat.completion', got '%s'", resp.Object)
	}
	if resp.Created != 1677858242 {
		t.Errorf("expected created 1677858242, got %d", resp.Created)
	}
	if resp.Model != "gpt-4-0613" {
		t.Errorf("expected model 'gpt-4-0613', got '%s'", resp.Model)
	}

	// Verify choices
	if len(resp.Choices) != 1 {
		t.Fatalf("expected 1 choice, got %d", len(resp.Choices))
	}
	choice := resp.Choices[0]
	if choice.Index != 0 {
		t.Errorf("expected choice index 0, got %d", choice.Index)
	}
	if choice.Message.Role != "assistant" {
		t.Errorf("expected message role 'assistant', got '%s'", choice.Message.Role)
	}
	if choice.Message.Content != "Hello! How can I assist you today?" {
		t.Errorf("unexpected message content: %s", choice.Message.Content)
	}
	if choice.FinishReason != "stop" {
		t.Errorf("expected finish_reason 'stop', got '%s'", choice.FinishReason)
	}

	// Verify usage
	if resp.Usage == nil {
		t.Fatal("expected usage to be set")
	}
	if resp.Usage.PromptTokens != 13 {
		t.Errorf("expected prompt_tokens 13, got %d", resp.Usage.PromptTokens)
	}
	if resp.Usage.CompletionTokens != 9 {
		t.Errorf("expected completion_tokens 9, got %d", resp.Usage.CompletionTokens)
	}
	if resp.Usage.TotalTokens != 22 {
		t.Errorf("expected total_tokens 22, got %d", resp.Usage.TotalTokens)
	}
}

func TestChatCompletionsResponse_DecodeWithoutUsage(t *testing.T) {
	// Test: Response decode when usage is not present (some streaming scenarios)
	input := `{
		"id": "chatcmpl-xyz789",
		"object": "chat.completion",
		"created": 1677858300,
		"model": "gpt-3.5-turbo",
		"choices": [
			{
				"index": 0,
				"message": {
					"role": "assistant",
					"content": "Response without usage stats."
				},
				"finish_reason": "stop"
			}
		]
	}`

	var resp ChatCompletionsResponse
	err := json.Unmarshal([]byte(input), &resp)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Verify usage is nil when not present
	if resp.Usage != nil {
		t.Errorf("expected usage to be nil, got %+v", resp.Usage)
	}

	// Verify other fields still parsed
	if resp.ID != "chatcmpl-xyz789" {
		t.Errorf("expected id 'chatcmpl-xyz789', got '%s'", resp.ID)
	}
	if len(resp.Choices) != 1 {
		t.Fatalf("expected 1 choice, got %d", len(resp.Choices))
	}
}

func TestChatCompletionsResponse_MultipleChoices(t *testing.T) {
	// Test: Response with multiple choices (n > 1)
	input := `{
		"id": "chatcmpl-multi",
		"object": "chat.completion",
		"created": 1677858500,
		"model": "gpt-4",
		"choices": [
			{
				"index": 0,
				"message": {"role": "assistant", "content": "First response"},
				"finish_reason": "stop"
			},
			{
				"index": 1,
				"message": {"role": "assistant", "content": "Second response"},
				"finish_reason": "stop"
			}
		],
		"usage": {
			"prompt_tokens": 10,
			"completion_tokens": 20,
			"total_tokens": 30
		}
	}`

	var resp ChatCompletionsResponse
	err := json.Unmarshal([]byte(input), &resp)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(resp.Choices) != 2 {
		t.Fatalf("expected 2 choices, got %d", len(resp.Choices))
	}

	if resp.Choices[0].Index != 0 || resp.Choices[0].Message.Content != "First response" {
		t.Errorf("unexpected first choice: %+v", resp.Choices[0])
	}
	if resp.Choices[1].Index != 1 || resp.Choices[1].Message.Content != "Second response" {
		t.Errorf("unexpected second choice: %+v", resp.Choices[1])
	}
}

func TestChatCompletionsResponse_UnknownFieldsIgnored(t *testing.T) {
	// Test: Response with unknown fields should decode without error
	// (response doesn't need to preserve unknown fields, just not crash)
	input := `{
		"id": "chatcmpl-unk",
		"object": "chat.completion",
		"created": 1677858600,
		"model": "gpt-4",
		"choices": [
			{
				"index": 0,
				"message": {"role": "assistant", "content": "Hello"},
				"finish_reason": "stop",
				"logprobs": null
			}
		],
		"usage": {
			"prompt_tokens": 5,
			"completion_tokens": 1,
			"total_tokens": 6
		},
		"system_fingerprint": "fp_abc123",
		"service_tier": "default"
	}`

	var resp ChatCompletionsResponse
	err := json.Unmarshal([]byte(input), &resp)
	if err != nil {
		t.Fatalf("unexpected error with unknown fields: %v", err)
	}

	// Verify known fields still work
	if resp.ID != "chatcmpl-unk" {
		t.Errorf("expected id 'chatcmpl-unk', got '%s'", resp.ID)
	}
	if len(resp.Choices) != 1 {
		t.Fatalf("expected 1 choice, got %d", len(resp.Choices))
	}
}

// ========================================================================
// Validation Tests
// ========================================================================

func TestChatCompletionsRequest_Validate_Success(t *testing.T) {
	// Test: Valid request passes validation
	temp := 0.7
	maxTokens := 100
	topP := 0.9

	req := ChatCompletionsRequest{
		Model: "gpt-4",
		Messages: []ChatMessage{
			{Role: "user", Content: "Hello"},
		},
		Temperature: &temp,
		MaxTokens:   &maxTokens,
		TopP:        &topP,
	}

	err := req.Validate()
	if err != nil {
		t.Errorf("expected validation to pass, got error: %v", err)
	}
}

func TestChatCompletionsRequest_Validate_MissingModel(t *testing.T) {
	// Test: Missing model fails validation
	req := ChatCompletionsRequest{
		Messages: []ChatMessage{
			{Role: "user", Content: "Hello"},
		},
	}

	err := req.Validate()
	if err == nil {
		t.Fatal("expected validation to fail for missing model")
	}
	if err.Error() != "model is required and must not be empty" {
		t.Errorf("unexpected error message: %v", err)
	}
}

func TestChatCompletionsRequest_Validate_EmptyModel(t *testing.T) {
	// Test: Empty model fails validation
	req := ChatCompletionsRequest{
		Model: "",
		Messages: []ChatMessage{
			{Role: "user", Content: "Hello"},
		},
	}

	err := req.Validate()
	if err == nil {
		t.Fatal("expected validation to fail for empty model")
	}
	if err.Error() != "model is required and must not be empty" {
		t.Errorf("unexpected error message: %v", err)
	}
}

func TestChatCompletionsRequest_Validate_MissingMessages(t *testing.T) {
	// Test: Missing messages fails validation
	req := ChatCompletionsRequest{
		Model: "gpt-4",
	}

	err := req.Validate()
	if err == nil {
		t.Fatal("expected validation to fail for missing messages")
	}
	if err.Error() != "messages is required and must contain at least one message" {
		t.Errorf("unexpected error message: %v", err)
	}
}

func TestChatCompletionsRequest_Validate_EmptyMessages(t *testing.T) {
	// Test: Empty messages array fails validation
	req := ChatCompletionsRequest{
		Model:    "gpt-4",
		Messages: []ChatMessage{},
	}

	err := req.Validate()
	if err == nil {
		t.Fatal("expected validation to fail for empty messages")
	}
	if err.Error() != "messages is required and must contain at least one message" {
		t.Errorf("unexpected error message: %v", err)
	}
}

func TestChatCompletionsRequest_Validate_MessageMissingRole(t *testing.T) {
	// Test: Message with empty role fails validation
	req := ChatCompletionsRequest{
		Model: "gpt-4",
		Messages: []ChatMessage{
			{Role: "", Content: "Hello"},
		},
	}

	err := req.Validate()
	if err == nil {
		t.Fatal("expected validation to fail for message with empty role")
	}
	if err.Error() != "message at index 0: role is required and must not be empty" {
		t.Errorf("unexpected error message: %v", err)
	}
}

func TestChatCompletionsRequest_Validate_MessageMissingContent(t *testing.T) {
	// Test: Message with empty content fails validation
	req := ChatCompletionsRequest{
		Model: "gpt-4",
		Messages: []ChatMessage{
			{Role: "user", Content: ""},
		},
	}

	err := req.Validate()
	if err == nil {
		t.Fatal("expected validation to fail for message with empty content")
	}
	if err.Error() != "message at index 0: content is required and must not be empty" {
		t.Errorf("unexpected error message: %v", err)
	}
}

func TestChatCompletionsRequest_Validate_TemperatureTooLow(t *testing.T) {
	// Test: Temperature below 0 fails validation
	temp := -0.1
	req := ChatCompletionsRequest{
		Model: "gpt-4",
		Messages: []ChatMessage{
			{Role: "user", Content: "Hello"},
		},
		Temperature: &temp,
	}

	err := req.Validate()
	if err == nil {
		t.Fatal("expected validation to fail for temperature < 0")
	}
	if err.Error() != "temperature must be between 0.0 and 2.0, got -0.100000" {
		t.Errorf("unexpected error message: %v", err)
	}
}

func TestChatCompletionsRequest_Validate_TemperatureTooHigh(t *testing.T) {
	// Test: Temperature above 2 fails validation
	temp := 2.1
	req := ChatCompletionsRequest{
		Model: "gpt-4",
		Messages: []ChatMessage{
			{Role: "user", Content: "Hello"},
		},
		Temperature: &temp,
	}

	err := req.Validate()
	if err == nil {
		t.Fatal("expected validation to fail for temperature > 2")
	}
	if err.Error() != "temperature must be between 0.0 and 2.0, got 2.100000" {
		t.Errorf("unexpected error message: %v", err)
	}
}

func TestChatCompletionsRequest_Validate_TopPTooLow(t *testing.T) {
	// Test: TopP below 0 fails validation
	topP := -0.1
	req := ChatCompletionsRequest{
		Model: "gpt-4",
		Messages: []ChatMessage{
			{Role: "user", Content: "Hello"},
		},
		TopP: &topP,
	}

	err := req.Validate()
	if err == nil {
		t.Fatal("expected validation to fail for top_p < 0")
	}
	if err.Error() != "top_p must be between 0.0 and 1.0, got -0.100000" {
		t.Errorf("unexpected error message: %v", err)
	}
}

func TestChatCompletionsRequest_Validate_TopPTooHigh(t *testing.T) {
	// Test: TopP above 1 fails validation
	topP := 1.1
	req := ChatCompletionsRequest{
		Model: "gpt-4",
		Messages: []ChatMessage{
			{Role: "user", Content: "Hello"},
		},
		TopP: &topP,
	}

	err := req.Validate()
	if err == nil {
		t.Fatal("expected validation to fail for top_p > 1")
	}
	if err.Error() != "top_p must be between 0.0 and 1.0, got 1.100000" {
		t.Errorf("unexpected error message: %v", err)
	}
}

func TestChatCompletionsRequest_Validate_MaxTokensNegative(t *testing.T) {
	// Test: Negative max_tokens fails validation
	maxTokens := -1
	req := ChatCompletionsRequest{
		Model: "gpt-4",
		Messages: []ChatMessage{
			{Role: "user", Content: "Hello"},
		},
		MaxTokens: &maxTokens,
	}

	err := req.Validate()
	if err == nil {
		t.Fatal("expected validation to fail for negative max_tokens")
	}
	if err.Error() != "max_tokens must be non-negative, got -1" {
		t.Errorf("unexpected error message: %v", err)
	}
}

func TestChatCompletionsRequest_Validate_BoundaryValues(t *testing.T) {
	// Test: Boundary values for temperature and top_p pass validation
	temp0 := 0.0
	temp2 := 2.0
	topP0 := 0.0
	topP1 := 1.0
	maxTokens0 := 0

	tests := []struct {
		name        string
		temperature *float64
		topP        *float64
		maxTokens   *int
	}{
		{"temperature 0.0", &temp0, nil, nil},
		{"temperature 2.0", &temp2, nil, nil},
		{"top_p 0.0", nil, &topP0, nil},
		{"top_p 1.0", nil, &topP1, nil},
		{"max_tokens 0", nil, nil, &maxTokens0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := ChatCompletionsRequest{
				Model: "gpt-4",
				Messages: []ChatMessage{
					{Role: "user", Content: "Hello"},
				},
				Temperature: tt.temperature,
				TopP:        tt.topP,
				MaxTokens:   tt.maxTokens,
			}

			err := req.Validate()
			if err != nil {
				t.Errorf("expected validation to pass for %s, got error: %v", tt.name, err)
			}
		})
	}
}

// ========================================================================
// MarshalJSON Empty Values Tests
// ========================================================================

func TestChatCompletionsRequest_MarshalEmptyModel(t *testing.T) {
	// Test: Empty model is not included in marshalled JSON
	req := ChatCompletionsRequest{
		Model: "",
		Messages: []ChatMessage{
			{Role: "user", Content: "Hello"},
		},
	}

	output, err := json.Marshal(req)
	if err != nil {
		t.Fatalf("unexpected error marshalling: %v", err)
	}

	var result map[string]any
	err = json.Unmarshal(output, &result)
	if err != nil {
		t.Fatalf("unexpected error unmarshalling result: %v", err)
	}

	// Empty model should not be in output
	if _, exists := result["model"]; exists {
		t.Errorf("expected empty model to be omitted from JSON, but found: %v", result["model"])
	}

	// Messages should still be present
	if _, exists := result["messages"]; !exists {
		t.Error("expected messages to be present in JSON")
	}
}

func TestChatCompletionsRequest_MarshalEmptyMessages(t *testing.T) {
	// Test: Empty messages array is not included in marshalled JSON
	req := ChatCompletionsRequest{
		Model:    "gpt-4",
		Messages: []ChatMessage{},
	}

	output, err := json.Marshal(req)
	if err != nil {
		t.Fatalf("unexpected error marshalling: %v", err)
	}

	var result map[string]any
	err = json.Unmarshal(output, &result)
	if err != nil {
		t.Fatalf("unexpected error unmarshalling result: %v", err)
	}

	// Empty messages should not be in output
	if _, exists := result["messages"]; exists {
		t.Errorf("expected empty messages to be omitted from JSON, but found: %v", result["messages"])
	}

	// Model should still be present
	if result["model"] != "gpt-4" {
		t.Errorf("expected model 'gpt-4', got %v", result["model"])
	}
}

func TestChatCompletionsRequest_MarshalNilMessages(t *testing.T) {
	// Test: Nil messages is not included in marshalled JSON
	req := ChatCompletionsRequest{
		Model:    "gpt-4",
		Messages: nil,
	}

	output, err := json.Marshal(req)
	if err != nil {
		t.Fatalf("unexpected error marshalling: %v", err)
	}

	var result map[string]any
	err = json.Unmarshal(output, &result)
	if err != nil {
		t.Fatalf("unexpected error unmarshalling result: %v", err)
	}

	// Nil messages should not be in output
	if _, exists := result["messages"]; exists {
		t.Errorf("expected nil messages to be omitted from JSON, but found: %v", result["messages"])
	}
}

func TestChatCompletionsRequest_MarshalValidFields(t *testing.T) {
	// Test: Non-empty required fields are included in marshalled JSON
	req := ChatCompletionsRequest{
		Model: "gpt-4",
		Messages: []ChatMessage{
			{Role: "user", Content: "Hello"},
		},
	}

	output, err := json.Marshal(req)
	if err != nil {
		t.Fatalf("unexpected error marshalling: %v", err)
	}

	var result map[string]any
	err = json.Unmarshal(output, &result)
	if err != nil {
		t.Fatalf("unexpected error unmarshalling result: %v", err)
	}

	// Both model and messages should be present
	if result["model"] != "gpt-4" {
		t.Errorf("expected model 'gpt-4', got %v", result["model"])
	}

	messages, ok := result["messages"].([]any)
	if !ok {
		t.Fatalf("expected messages to be an array, got %T", result["messages"])
	}
	if len(messages) != 1 {
		t.Errorf("expected 1 message, got %d", len(messages))
	}
}

// ========================================================================
// Error Context Tests
// ========================================================================

func TestChatCompletionsRequest_UnmarshalJSON_MalformedJSON(t *testing.T) {
	// Test: Malformed JSON returns error
	input := `{"model": "gpt-4", "messages": [`

	var req ChatCompletionsRequest
	err := json.Unmarshal([]byte(input), &req)
	if err == nil {
		t.Fatal("expected error for malformed JSON")
	}
	// Error should come from initial unmarshal, not have field context
}

func TestChatCompletionsRequest_UnmarshalJSON_InvalidTypeForModel(t *testing.T) {
	// Test: Invalid type for model field returns error with context
	input := `{
		"model": 12345,
		"messages": [{"role": "user", "content": "Hello"}]
	}`

	var req ChatCompletionsRequest
	err := json.Unmarshal([]byte(input), &req)
	if err == nil {
		t.Fatal("expected error for invalid model type")
	}

	// Error should contain field name 'model'
	if !strings.Contains(err.Error(), "model") {
		t.Errorf("expected error to mention field 'model', got: %v", err)
	}
}

func TestChatCompletionsRequest_UnmarshalJSON_InvalidTypeForTemperature(t *testing.T) {
	// Test: Invalid type for temperature field returns error with context
	input := `{
		"model": "gpt-4",
		"messages": [{"role": "user", "content": "Hello"}],
		"temperature": "not-a-number"
	}`

	var req ChatCompletionsRequest
	err := json.Unmarshal([]byte(input), &req)
	if err == nil {
		t.Fatal("expected error for invalid temperature type")
	}

	// Error should contain field name 'temperature'
	if !strings.Contains(err.Error(), "temperature") {
		t.Errorf("expected error to mention field 'temperature', got: %v", err)
	}
}

func TestChatCompletionsRequest_UnmarshalJSON_InvalidTypeForMessages(t *testing.T) {
	// Test: Invalid type for messages field returns error with context
	input := `{
		"model": "gpt-4",
		"messages": "not-an-array"
	}`

	var req ChatCompletionsRequest
	err := json.Unmarshal([]byte(input), &req)
	if err == nil {
		t.Fatal("expected error for invalid messages type")
	}

	// Error should contain field name 'messages'
	if !strings.Contains(err.Error(), "messages") {
		t.Errorf("expected error to mention field 'messages', got: %v", err)
	}
}

func TestChatCompletionsRequest_UnmarshalJSON_InvalidTypeForStream(t *testing.T) {
	// Test: Invalid type for stream field returns error with context
	input := `{
		"model": "gpt-4",
		"messages": [{"role": "user", "content": "Hello"}],
		"stream": "not-a-bool"
	}`

	var req ChatCompletionsRequest
	err := json.Unmarshal([]byte(input), &req)
	if err == nil {
		t.Fatal("expected error for invalid stream type")
	}

	// Error should contain field name 'stream'
	if !strings.Contains(err.Error(), "stream") {
		t.Errorf("expected error to mention field 'stream', got: %v", err)
	}
}

func TestChatCompletionsRequest_UnmarshalJSON_InvalidTypeForMaxTokens(t *testing.T) {
	// Test: Invalid type for max_tokens field returns error with context
	input := `{
		"model": "gpt-4",
		"messages": [{"role": "user", "content": "Hello"}],
		"max_tokens": "not-a-number"
	}`

	var req ChatCompletionsRequest
	err := json.Unmarshal([]byte(input), &req)
	if err == nil {
		t.Fatal("expected error for invalid max_tokens type")
	}

	// Error should contain field name 'max_tokens'
	if !strings.Contains(err.Error(), "max_tokens") {
		t.Errorf("expected error to mention field 'max_tokens', got: %v", err)
	}
}

func TestChatCompletionsRequest_UnmarshalJSON_InvalidTypeForTopP(t *testing.T) {
	// Test: Invalid type for top_p field returns error with context
	input := `{
		"model": "gpt-4",
		"messages": [{"role": "user", "content": "Hello"}],
		"top_p": "not-a-number"
	}`

	var req ChatCompletionsRequest
	err := json.Unmarshal([]byte(input), &req)
	if err == nil {
		t.Fatal("expected error for invalid top_p type")
	}

	// Error should contain field name 'top_p'
	if !strings.Contains(err.Error(), "top_p") {
		t.Errorf("expected error to mention field 'top_p', got: %v", err)
	}
}

func TestChatCompletionsRequest_UnmarshalJSON_InvalidExtraField(t *testing.T) {
	// Test: This test verifies that we successfully parse extra fields
	// Note: It's difficult to create a JSON that parses as RawMessage but
	// fails when unmarshalling to 'any', as both are very permissive.
	// This test exists as a placeholder for edge cases.
	input := `{
		"model": "gpt-4",
		"messages": [{"role": "user", "content": "Hello"}],
		"custom_field": {"nested": "value"}
	}`

	var req ChatCompletionsRequest
	err := json.Unmarshal([]byte(input), &req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Verify the extra field was captured
	if req.Extra == nil {
		t.Fatal("expected Extra to be non-nil")
	}
	if _, ok := req.Extra["custom_field"]; !ok {
		t.Error("expected custom_field in Extra")
	}
}
