package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"time"

	"navplane/internal/config"
)

const (
	requestTimeout     = 5 * time.Minute
	maxRequestBodySize = 10 * 1024 * 1024 // 10 MB
)

// chatCompletionsHandler handles POST /v1/chat/completions as a passthrough proxy.
//
// Design goals:
//  1. Full transparency: Upstream responses (including errors) returned as-is
//  2. No request validation: Upstream provider validates the request
//  3. Minimal parsing: Only check stream flag for routing
//
// NavPlane errors only for: 405, 400 (read fail), 413, 501 (streaming), 502, 504
type chatCompletionsHandler struct {
	upstreamURL string
	apiKey      string
	client      *http.Client
}

func newHandler(cfg *config.Config, client *http.Client) *chatCompletionsHandler {
	if client == nil {
		client = &http.Client{
			Timeout: 0,
			CheckRedirect: func(req *http.Request, via []*http.Request) error {
				return http.ErrUseLastResponse
			},
		}
	}
	return &chatCompletionsHandler{
		upstreamURL: strings.TrimSuffix(cfg.Provider.BaseURL, "/") + "/v1/chat/completions",
		apiKey:      cfg.Provider.APIKey,
		client:      client,
	}
}

func (h *chatCompletionsHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// Defense-in-depth: mux routes by method, but check here for direct handler use
	if r.Method != http.MethodPost {
		writeProxyError(w, http.StatusMethodNotAllowed, "method not allowed", "invalid_request_error")
		return
	}

	defer r.Body.Close()

	body, err := io.ReadAll(io.LimitReader(r.Body, maxRequestBodySize+1))
	if err != nil {
		writeProxyError(w, http.StatusBadRequest, "failed to read request body", "invalid_request_error")
		return
	}
	if len(body) > maxRequestBodySize {
		writeProxyError(w, http.StatusRequestEntityTooLarge, "request body too large", "invalid_request_error")
		return
	}

	if isStreamingRequest(body) {
		writeProxyError(w, http.StatusNotImplemented, "streaming not implemented yet", "not_implemented_error")
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), requestTimeout)
	defer cancel()

	upstreamReq, err := http.NewRequestWithContext(ctx, http.MethodPost, h.upstreamURL, bytes.NewReader(body))
	if err != nil {
		writeProxyError(w, http.StatusInternalServerError, "failed to create upstream request", "server_error")
		return
	}

	setUpstreamHeaders(upstreamReq, r, h.apiKey)

	upstreamResp, err := h.client.Do(upstreamReq)
	if err != nil {
		if r.Context().Err() == context.Canceled {
			return // Client disconnected
		}
		if ctx.Err() == context.DeadlineExceeded {
			writeProxyError(w, http.StatusGatewayTimeout, "upstream request timed out", "server_error")
			return
		}
		writeProxyError(w, http.StatusBadGateway, "failed to reach upstream provider", "server_error")
		return
	}
	defer upstreamResp.Body.Close()

	copyResponseHeaders(w, upstreamResp)
	w.WriteHeader(upstreamResp.StatusCode)
	io.Copy(w, upstreamResp.Body) // Error ignored: can't send error after headers written
}

func isStreamingRequest(body []byte) bool {
	var partial struct {
		Stream *bool `json:"stream"`
	}
	if err := json.Unmarshal(body, &partial); err != nil {
		return false
	}
	return partial.Stream != nil && *partial.Stream
}

func setUpstreamHeaders(upstream *http.Request, original *http.Request, apiKey string) {
	upstream.Header.Set("Content-Type", "application/json")
	upstream.Header.Set("User-Agent", "NavPlane/1.0")
	// SECURITY: Always use provider key, never forward client auth
	upstream.Header.Set("Authorization", "Bearer "+apiKey)

	if v := original.Header.Get("Accept"); v != "" {
		upstream.Header.Set("Accept", v)
	}
	if v := original.Header.Get("OpenAI-Organization"); v != "" {
		upstream.Header.Set("OpenAI-Organization", v)
	}
}

func copyResponseHeaders(w http.ResponseWriter, resp *http.Response) {
	headers := []string{
		"Content-Type",
		"Content-Length",
		"Content-Encoding",
		"X-Request-Id",
		"X-RateLimit-Limit-Requests",
		"X-RateLimit-Limit-Tokens",
		"X-RateLimit-Remaining-Requests",
		"X-RateLimit-Remaining-Tokens",
		"X-RateLimit-Reset-Requests",
		"X-RateLimit-Reset-Tokens",
	}
	for _, h := range headers {
		if v := resp.Header.Get(h); v != "" {
			w.Header().Set(h, v)
		}
	}
}

func writeProxyError(w http.ResponseWriter, statusCode int, message, errorType string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(map[string]any{
		"error": map[string]any{
			"message": message,
			"type":    errorType,
		},
	})
}

// NewChatCompletionsHandler creates a handler for production use.
func NewChatCompletionsHandler(cfg *config.Config) http.HandlerFunc {
	return newHandler(cfg, nil).ServeHTTP
}

// NewChatCompletionsHandlerWithClient creates a handler with custom HTTP client (for testing).
func NewChatCompletionsHandlerWithClient(cfg *config.Config, client *http.Client) http.HandlerFunc {
	return newHandler(cfg, client).ServeHTTP
}
