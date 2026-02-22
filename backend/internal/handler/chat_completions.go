package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"log"
	"net/http"
	"strings"
	"time"

	"navplane/internal/middleware"
	"navplane/internal/provider"
	"navplane/internal/providerkey"

	"github.com/google/uuid"
)

const (
	requestTimeout     = 5 * time.Minute
	maxRequestBodySize = 10 * 1024 * 1024 // 10 MB
)

func closeBody(body io.Closer) {
	if err := body.Close(); err != nil {
		log.Printf("failed to close body: %v", err)
	}
}

// ProviderKeyGetter is the interface for getting decrypted provider keys.
type ProviderKeyGetter interface {
	GetDecryptedKey(ctx context.Context, orgID uuid.UUID, providerName string) (string, error)
}

// ChatCompletionsDeps holds dependencies for the chat completions handler.
type ChatCompletionsDeps struct {
	ProviderRegistry   *provider.Registry
	ProviderKeyManager ProviderKeyGetter
}

// chatCompletionsHandler handles POST /v1/chat/completions as a passthrough proxy.
//
// Design goals:
//  1. Full transparency: Upstream responses (including errors) returned as-is
//  2. No request validation: Upstream provider validates the request
//  3. Minimal parsing: Only check stream flag and model for routing
//  4. SSE streaming: Stream responses with continuous flushing when stream=true
//
// NavPlane errors only for: 405, 400 (read fail), 413, 502, 504
type chatCompletionsHandler struct {
	providerRegistry   *provider.Registry
	providerKeyManager ProviderKeyGetter
	client             *http.Client
}

func newChatHandler(deps *ChatCompletionsDeps, client *http.Client) *chatCompletionsHandler {
	if client == nil {
		client = &http.Client{
			Timeout: 0, // Per-request timeout via context
			CheckRedirect: func(req *http.Request, via []*http.Request) error {
				return http.ErrUseLastResponse
			},
		}
	}

	return &chatCompletionsHandler{
		providerRegistry:   deps.ProviderRegistry,
		providerKeyManager: deps.ProviderKeyManager,
		client:             client,
	}
}

// requestInfo holds parsed request metadata.
type requestInfo struct {
	Model        string
	ProviderName string
	IsStreaming  bool
}

func (h *chatCompletionsHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// Defense-in-depth: mux routes by method, but check here for direct handler use
	if r.Method != http.MethodPost {
		writeProxyError(w, http.StatusMethodNotAllowed, "method not allowed", "invalid_request_error")
		return
	}

	// Get org from context (set by auth middleware)
	org := middleware.GetOrg(r.Context())
	if org == nil {
		writeProxyError(w, http.StatusUnauthorized, "unauthorized", "authentication_error")
		return
	}

	defer closeBody(r.Body)

	body, err := io.ReadAll(io.LimitReader(r.Body, maxRequestBodySize+1))
	if err != nil {
		writeProxyError(w, http.StatusBadRequest, "failed to read request body", "invalid_request_error")
		return
	}
	if len(body) > maxRequestBodySize {
		writeProxyError(w, http.StatusRequestEntityTooLarge, "request body too large", "invalid_request_error")
		return
	}

	// Parse request to determine provider and streaming mode
	reqInfo, err := h.parseRequest(body)
	if err != nil {
		writeProxyError(w, http.StatusBadRequest, err.Error(), "invalid_request_error")
		return
	}

	// Get provider
	p, err := h.providerRegistry.Get(reqInfo.ProviderName)
	if err != nil {
		writeProxyError(w, http.StatusBadRequest, "unsupported provider for model: "+reqInfo.Model, "invalid_request_error")
		return
	}

	// Get decrypted provider key for this org
	apiKey, err := h.providerKeyManager.GetDecryptedKey(r.Context(), org.ID, reqInfo.ProviderName)
	if err != nil {
		if err == providerkey.ErrNotFound {
			writeProxyError(w, http.StatusBadRequest, "no API key configured for provider: "+p.DisplayName(), "invalid_request_error")
			return
		}
		log.Printf("failed to get provider key: %v", err)
		writeProxyError(w, http.StatusInternalServerError, "failed to retrieve provider credentials", "server_error")
		return
	}

	// Build upstream URL
	upstreamURL := strings.TrimSuffix(p.BaseURL(), "/") + "/chat/completions"

	if reqInfo.IsStreaming {
		h.handleStreaming(w, r, body, upstreamURL, p, apiKey)
	} else {
		h.handleNonStreaming(w, r, body, upstreamURL, p, apiKey)
	}
}

// parseRequest extracts model and streaming info from the request body.
func (h *chatCompletionsHandler) parseRequest(body []byte) (*requestInfo, error) {
	var partial struct {
		Model  string `json:"model"`
		Stream *bool  `json:"stream"`
	}
	if err := json.Unmarshal(body, &partial); err != nil {
		return nil, err
	}

	if partial.Model == "" {
		return nil, &jsonError{Message: "model is required"}
	}

	// Determine provider from model
	providerName := h.determineProvider(partial.Model)

	return &requestInfo{
		Model:        partial.Model,
		ProviderName: providerName,
		IsStreaming:  partial.Stream != nil && *partial.Stream,
	}, nil
}

// determineProvider determines the provider from the model name.
func (h *chatCompletionsHandler) determineProvider(model string) string {
	model = strings.ToLower(model)

	// OpenAI models
	if strings.HasPrefix(model, "gpt-") || strings.HasPrefix(model, "o1") {
		return "openai"
	}

	// Anthropic models
	if strings.HasPrefix(model, "claude-") {
		return "anthropic"
	}

	// Default to OpenAI for unknown models
	return "openai"
}

type jsonError struct {
	Message string
}

func (e *jsonError) Error() string {
	return e.Message
}

func (h *chatCompletionsHandler) handleNonStreaming(w http.ResponseWriter, r *http.Request, body []byte, upstreamURL string, p provider.Provider, apiKey string) {
	start := time.Now()
	reqID := r.Header.Get("X-Request-ID")

	ctx, cancel := context.WithTimeout(r.Context(), requestTimeout)
	defer cancel()

	upstreamReq, err := http.NewRequestWithContext(ctx, http.MethodPost, upstreamURL, bytes.NewReader(body))
	if err != nil {
		writeProxyError(w, http.StatusInternalServerError, "failed to create upstream request", "server_error")
		return
	}

	setUpstreamHeaders(upstreamReq, r, p, apiKey)

	upstreamResp, err := h.client.Do(upstreamReq)
	if err != nil {
		if r.Context().Err() == context.Canceled {
			return
		}
		if ctx.Err() == context.DeadlineExceeded {
			writeProxyError(w, http.StatusGatewayTimeout, "upstream request timed out", "server_error")
			logRequest(r.URL.Path, http.StatusGatewayTimeout, time.Since(start), reqID)
			return
		}
		writeProxyError(w, http.StatusBadGateway, "failed to reach upstream provider", "server_error")
		logRequest(r.URL.Path, http.StatusBadGateway, time.Since(start), reqID)
		return
	}
	defer closeBody(upstreamResp.Body)

	copyResponseHeaders(w, upstreamResp)
	w.WriteHeader(upstreamResp.StatusCode)
	if _, err := io.Copy(w, upstreamResp.Body); err != nil {
		log.Printf("failed to copy upstream response: %v", err)
	}

	logRequest(r.URL.Path, upstreamResp.StatusCode, time.Since(start), reqID)
}

func (h *chatCompletionsHandler) handleStreaming(w http.ResponseWriter, r *http.Request, body []byte, upstreamURL string, p provider.Provider, apiKey string) {
	start := time.Now()
	reqID := r.Header.Get("X-Request-ID")

	// No timeout for streaming - runs until upstream closes or client disconnects
	ctx, cancel := context.WithCancel(r.Context())
	defer cancel()

	upstreamReq, err := http.NewRequestWithContext(ctx, http.MethodPost, upstreamURL, bytes.NewReader(body))
	if err != nil {
		writeProxyError(w, http.StatusInternalServerError, "failed to create upstream request", "server_error")
		return
	}

	setUpstreamHeaders(upstreamReq, r, p, apiKey)
	upstreamReq.Header.Set("Accept", "text/event-stream")

	upstreamResp, err := h.client.Do(upstreamReq)
	if err != nil {
		if ctx.Err() != nil {
			return // Client disconnected
		}
		writeProxyError(w, http.StatusBadGateway, "failed to reach upstream provider", "server_error")
		logRequest(r.URL.Path, http.StatusBadGateway, time.Since(start), reqID)
		return
	}
	defer closeBody(upstreamResp.Body)

	// Non-200: pass through as regular response (not SSE)
	if upstreamResp.StatusCode != http.StatusOK {
		upstreamBody, err := io.ReadAll(upstreamResp.Body)
		if err != nil {
			writeProxyError(w, http.StatusBadGateway, "failed to read upstream error response", "server_error")
			logRequest(r.URL.Path, http.StatusBadGateway, time.Since(start), reqID)
			return
		}
		copyResponseHeaders(w, upstreamResp)
		w.WriteHeader(upstreamResp.StatusCode)
		if _, err := w.Write(upstreamBody); err != nil {
			log.Printf("failed to write upstream error response: %v", err)
		}
		logRequest(r.URL.Path, upstreamResp.StatusCode, time.Since(start), reqID)
		return
	}

	// Check if we can flush
	flusher, ok := w.(http.Flusher)
	if !ok {
		writeProxyError(w, http.StatusInternalServerError, "streaming not supported", "server_error")
		return
	}

	// Copy rate limit headers from upstream before setting SSE headers
	copyRateLimitHeaders(w, upstreamResp)

	// Set SSE headers (these override Content-Type from upstream)
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("X-Accel-Buffering", "no")
	flusher.Flush()

	// Stream response body
	buf := make([]byte, 4096)
	for {
		n, err := upstreamResp.Body.Read(buf)
		if n > 0 {
			if _, writeErr := w.Write(buf[:n]); writeErr != nil {
				return // Client disconnected
			}
			flusher.Flush()
		}
		if err != nil {
			break // EOF or error (including context cancellation)
		}
	}

	logRequest(r.URL.Path, http.StatusOK, time.Since(start), reqID)
}

func setUpstreamHeaders(upstream *http.Request, original *http.Request, p provider.Provider, apiKey string) {
	upstream.Header.Set("Content-Type", "application/json")
	upstream.Header.Set("User-Agent", "NavPlane/1.0")

	// Set provider-specific auth header
	upstream.Header.Set(p.AuthHeader(), p.FormatAuthValue(apiKey))

	// Anthropic requires version header
	if p.Name() == "anthropic" {
		upstream.Header.Set("anthropic-version", "2023-06-01")
	}

	if v := original.Header.Get("Accept"); v != "" {
		upstream.Header.Set("Accept", v)
	}
	if v := original.Header.Get("OpenAI-Organization"); v != "" {
		upstream.Header.Set("OpenAI-Organization", v)
	}
	if v := original.Header.Get("X-Request-ID"); v != "" {
		upstream.Header.Set("X-Request-ID", v)
	}
}

func copyResponseHeaders(w http.ResponseWriter, resp *http.Response) {
	headers := []string{
		"Content-Type",
		"Content-Length",
		"Content-Encoding",
		"X-Request-Id",
	}
	for _, h := range headers {
		if v := resp.Header.Get(h); v != "" {
			w.Header().Set(h, v)
		}
	}
	copyRateLimitHeaders(w, resp)
}

func copyRateLimitHeaders(w http.ResponseWriter, resp *http.Response) {
	rateLimitHeaders := []string{
		"X-RateLimit-Limit-Requests",
		"X-RateLimit-Limit-Tokens",
		"X-RateLimit-Remaining-Requests",
		"X-RateLimit-Remaining-Tokens",
		"X-RateLimit-Reset-Requests",
		"X-RateLimit-Reset-Tokens",
	}
	for _, h := range rateLimitHeaders {
		if v := resp.Header.Get(h); v != "" {
			w.Header().Set(h, v)
		}
	}
}

func writeProxyError(w http.ResponseWriter, statusCode int, message, errorType string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	if err := json.NewEncoder(w).Encode(map[string]any{
		"error": map[string]any{
			"message": message,
			"type":    errorType,
		},
	}); err != nil {
		log.Printf("failed to write proxy error response: %v", err)
	}
}

func logRequest(path string, status int, duration time.Duration, reqID string) {
	if reqID != "" {
		log.Printf("route=%s status=%d duration=%s request_id=%s", path, status, duration, reqID)
	} else {
		log.Printf("route=%s status=%d duration=%s", path, status, duration)
	}
}

// NewChatCompletionsHandler creates a handler for production use.
func NewChatCompletionsHandler(deps *ChatCompletionsDeps) http.HandlerFunc {
	return newChatHandler(deps, nil).ServeHTTP
}

// NewChatCompletionsHandlerWithClient creates a handler with custom HTTP client (for testing).
func NewChatCompletionsHandlerWithClient(deps *ChatCompletionsDeps, client *http.Client) http.HandlerFunc {
	return newChatHandler(deps, client).ServeHTTP
}
