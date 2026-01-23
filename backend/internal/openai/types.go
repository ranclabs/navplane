// Package openai provides request/response contracts for OpenAI-compatible API endpoints.
// This package implements the MVP subset of the chat completions API, focusing on
// compatibility and resilience rather than full coverage.
package openai

import (
	"encoding/json"
	"errors"
	"fmt"
)

// ChatMessage represents a single message in a chat conversation.
// For MVP, content is a plain string only (no multimodal/array content support).
type ChatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// ChatCompletionsRequest represents the request body for POST /v1/chat/completions.
//
// Unknown fields handling:
// This struct implements custom JSON unmarshalling to capture any fields not explicitly
// defined in the struct. Unknown fields are stored in the Extra map, allowing the handler
// to forward the original payload to upstream providers without losing vendor-specific
// or newer API fields that we don't explicitly support yet.
//
// Example unknown fields that might be passed through: response_format, tools, metadata,
// seed, logprobs, etc. These won't break parsing and will be available in Extra.
type ChatCompletionsRequest struct {
	// Model is the ID of the model to use (e.g., "gpt-4", "gpt-3.5-turbo").
	Model string `json:"model"`

	// Messages is the list of messages comprising the conversation so far.
	Messages []ChatMessage `json:"messages"`

	// Stream indicates whether to stream partial message deltas.
	// Pointer type distinguishes unset from explicit false.
	Stream *bool `json:"stream,omitempty"`

	// Temperature controls randomness in output (0.0-2.0).
	// Pointer type distinguishes unset from explicit 0.
	Temperature *float64 `json:"temperature,omitempty"`

	// MaxTokens limits the maximum number of tokens to generate.
	// Pointer type distinguishes unset from explicit 0.
	MaxTokens *int `json:"max_tokens,omitempty"`

	// TopP controls nucleus sampling probability mass (0.0-1.0).
	// Pointer type distinguishes unset from explicit 0.
	TopP *float64 `json:"top_p,omitempty"`

	// Extra holds any unknown fields from the original JSON payload.
	// This enables passthrough of vendor-specific or newer API fields
	// that we don't explicitly support, ensuring forward compatibility.
	// Use MarshalJSON to include these fields when forwarding the request.
	Extra map[string]any `json:"-"`
}

// Validate checks that the request has all required fields and valid values.
// This should be called by handlers before forwarding requests to providers.
func (r *ChatCompletionsRequest) Validate() error {
	if r.Model == "" {
		return errors.New("model is required and must not be empty")
	}

	if len(r.Messages) == 0 {
		return errors.New("messages is required and must contain at least one message")
	}

	// Validate each message has role and content
	for i, msg := range r.Messages {
		if msg.Role == "" {
			return fmt.Errorf("message at index %d: role is required and must not be empty", i)
		}
		if msg.Content == "" {
			return fmt.Errorf("message at index %d: content is required and must not be empty", i)
		}
	}

	// Validate temperature range if set
	if r.Temperature != nil {
		if *r.Temperature < 0 || *r.Temperature > 2 {
			return fmt.Errorf("temperature must be between 0.0 and 2.0, got %f", *r.Temperature)
		}
	}

	// Validate top_p range if set
	if r.TopP != nil {
		if *r.TopP < 0 || *r.TopP > 1 {
			return fmt.Errorf("top_p must be between 0.0 and 1.0, got %f", *r.TopP)
		}
	}

	// Validate max_tokens is non-negative if set
	if r.MaxTokens != nil {
		if *r.MaxTokens < 0 {
			return fmt.Errorf("max_tokens must be non-negative, got %d", *r.MaxTokens)
		}
	}

	return nil
}

// knownRequestFields is the set of field names we explicitly handle.
// Used during unmarshalling to identify which fields go into Extra.
var knownRequestFields = map[string]bool{
	"model":       true,
	"messages":    true,
	"stream":      true,
	"temperature": true,
	"max_tokens":  true,
	"top_p":       true,
}

// UnmarshalJSON implements custom JSON unmarshalling for ChatCompletionsRequest.
// It unmarshals into a raw map once, then selectively unmarshals known fields
// from the map and stores unknown fields in Extra.
// This approach ensures we never lose data from the original request.
func (r *ChatCompletionsRequest) UnmarshalJSON(data []byte) error {
	// Unmarshal everything into a raw map (single parse)
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}

	// Unmarshal known fields from the raw map
	if modelRaw, ok := raw["model"]; ok {
		if err := json.Unmarshal(modelRaw, &r.Model); err != nil {
			return fmt.Errorf("unmarshalling field 'model': %w", err)
		}
	}

	if messagesRaw, ok := raw["messages"]; ok {
		if err := json.Unmarshal(messagesRaw, &r.Messages); err != nil {
			return fmt.Errorf("unmarshalling field 'messages': %w", err)
		}
	}

	if streamRaw, ok := raw["stream"]; ok {
		if err := json.Unmarshal(streamRaw, &r.Stream); err != nil {
			return fmt.Errorf("unmarshalling field 'stream': %w", err)
		}
	}

	if tempRaw, ok := raw["temperature"]; ok {
		if err := json.Unmarshal(tempRaw, &r.Temperature); err != nil {
			return fmt.Errorf("unmarshalling field 'temperature': %w", err)
		}
	}

	if maxTokensRaw, ok := raw["max_tokens"]; ok {
		if err := json.Unmarshal(maxTokensRaw, &r.MaxTokens); err != nil {
			return fmt.Errorf("unmarshalling field 'max_tokens': %w", err)
		}
	}

	if topPRaw, ok := raw["top_p"]; ok {
		if err := json.Unmarshal(topPRaw, &r.TopP); err != nil {
			return fmt.Errorf("unmarshalling field 'top_p': %w", err)
		}
	}

	// Extract unknown fields into Extra
	r.Extra = make(map[string]any)
	for key, rawValue := range raw {
		if !knownRequestFields[key] {
			var value any
			if err := json.Unmarshal(rawValue, &value); err != nil {
				return fmt.Errorf("unmarshalling extra field %q: %w", key, err)
			}
			r.Extra[key] = value
		}
	}

	// If no unknown fields, set Extra to nil to keep it clean
	if len(r.Extra) == 0 {
		r.Extra = nil
	}

	return nil
}

// MarshalJSON implements custom JSON marshalling for ChatCompletionsRequest.
// It includes both the known fields and any fields stored in Extra,
// enabling full-fidelity forwarding of the original request.
func (r ChatCompletionsRequest) MarshalJSON() ([]byte, error) {
	// Start with Extra fields if present
	result := make(map[string]any)
	for k, v := range r.Extra {
		result[k] = v
	}

	// Add known fields (these take precedence over Extra)
	// Only include required fields if they are non-empty
	if r.Model != "" {
		result["model"] = r.Model
	}
	if len(r.Messages) > 0 {
		result["messages"] = r.Messages
	}

	// Optional fields - only include if set
	if r.Stream != nil {
		result["stream"] = *r.Stream
	}
	if r.Temperature != nil {
		result["temperature"] = *r.Temperature
	}
	if r.MaxTokens != nil {
		result["max_tokens"] = *r.MaxTokens
	}
	if r.TopP != nil {
		result["top_p"] = *r.TopP
	}

	return json.Marshal(result)
}

// ChatCompletionsResponse represents the response from POST /v1/chat/completions.
// This handles non-streaming responses only for MVP.
type ChatCompletionsResponse struct {
	// ID is a unique identifier for the completion.
	ID string `json:"id"`

	// Object is the object type, always "chat.completion" for non-streaming.
	Object string `json:"object"`

	// Created is the Unix timestamp of when the completion was created.
	Created int64 `json:"created"`

	// Model is the model used for the completion.
	Model string `json:"model"`

	// Choices is the list of completion choices.
	Choices []Choice `json:"choices"`

	// Usage contains token usage statistics (optional, may be nil).
	Usage *Usage `json:"usage,omitempty"`
}

// Choice represents a single completion choice in the response.
type Choice struct {
	// Index is the index of this choice in the list.
	Index int `json:"index"`

	// Message is the generated message.
	Message ChatMessage `json:"message"`

	// FinishReason indicates why the model stopped generating.
	// Common values: "stop", "length", "content_filter", "tool_calls".
	FinishReason string `json:"finish_reason"`
}

// Usage contains token usage statistics for a completion request.
type Usage struct {
	// PromptTokens is the number of tokens in the prompt.
	PromptTokens int `json:"prompt_tokens"`

	// CompletionTokens is the number of tokens in the completion.
	CompletionTokens int `json:"completion_tokens"`

	// TotalTokens is the total number of tokens used.
	TotalTokens int `json:"total_tokens"`
}
