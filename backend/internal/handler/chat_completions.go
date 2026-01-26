package handler

import (
	"encoding/json"
	"net/http"

	"navplane/internal/openai"
)

// ChatCompletions handles POST /v1/chat/completions requests.
// This is the OpenAI-compatible chat completions endpoint.
//
// Current behavior (MVP):
//   - Parses and validates the request body
//   - Returns 200 OK with placeholder response if valid
//   - Returns 400 Bad Request for invalid JSON or validation errors
//   - Returns 405 Method Not Allowed for non-POST requests
//
// Note: Upstream forwarding will be implemented in a future task.
// Request bodies are intentionally not logged for security.
func ChatCompletions(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	// Only allow POST method
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		json.NewEncoder(w).Encode(map[string]any{
			"error": map[string]any{
				"message": "method not allowed",
				"type":    "invalid_request_error",
			},
		})
		return
	}

	// Parse the request body
	var req openai.ChatCompletionsRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]any{
			"error": map[string]any{
				"message": "invalid JSON: " + err.Error(),
				"type":    "invalid_request_error",
			},
		})
		return
	}

	// Validate the request
	if err := req.Validate(); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]any{
			"error": map[string]any{
				"message": err.Error(),
				"type":    "invalid_request_error",
			},
		})
		return
	}

	// Placeholder success response (until upstream forwarding is implemented)
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]any{
		"status": "ok",
	})
}
