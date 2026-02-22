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
//  4. SSE streaming: Stream responses with continuous flushing when stream=true
//
// NavPlane errors only for: 405, 400 (read fail), 413, 502, 504
type chatCompletionsHandler struct {
	upstreamURL string
	apiKey      string
	client      *http.Client
}

func newHandler(cfg *config.Config, client *http.Client) *chatCompletionsHandler {
	if client == nil {
		client = &http.Client{
			Timeout: 0, // Per-request timeout via context
			CheckRedirect: func(req *http.Request, via []*http.Request) error {
				return http.ErrUseLastResponse
			},
		}
	}
	// Normalize base URL: strip trailing slash and /v1 suffix to avoid duplication
	baseURL := strings.TrimSuffix(cfg.Provider.BaseURL, "/")
	baseURL = strings.TrimSuffix(baseURL, "/v1")

	return &chatCompletionsHandler{
		upstreamURL: baseURL + "/v1/chat/completions",
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
		h.handleStreaming(w, r, body)
	} else {
		h.handleNonStreaming(w, r, body)
	}
}

func (h *chatCompletionsHandler) handleNonStreaming(w http.ResponseWriter, r *http.Request, body []byte) {
	start := time.Now()
	reqID := r.Header.Get("X-Request-ID")

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
	defer upstreamResp.Body.Close()

	copyResponseHeaders(w, upstreamResp)
	w.WriteHeader(upstreamResp.StatusCode)
	_, _ = io.Copy(w, upstreamResp.Body)

	logRequest(r.URL.Path, upstreamResp.StatusCode, time.Since(start), reqID)
}

func (h *chatCompletionsHandler) handleStreaming(w http.ResponseWriter, r *http.Request, body []byte) {
	start := time.Now()
	reqID := r.Header.Get("X-Request-ID")

	// No timeout for streaming - runs until upstream closes or client disconnects
	ctx, cancel := context.WithCancel(r.Context())
	defer cancel()

	upstreamReq, err := http.NewRequestWithContext(ctx, http.MethodPost, h.upstreamURL, bytes.NewReader(body))
	if err != nil {
		writeProxyError(w, http.StatusInternalServerError, "failed to create upstream request", "server_error")
		return
	}

	setUpstreamHeaders(upstreamReq, r, h.apiKey)
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
	defer upstreamResp.Body.Close()

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
		_, _ = w.Write(upstreamBody)
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
	// Note: Context cancellation closes the HTTP connection, causing Read to return an error.
	// No explicit select needed - the transport layer handles cancellation.
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
	_ = json.NewEncoder(w).Encode(map[string]any{
		"error": map[string]any{
			"message": message,
			"type":    errorType,
		},
	})
}

func logRequest(path string, status int, duration time.Duration, reqID string) {
	if reqID != "" {
		log.Printf("route=%s status=%d duration=%s request_id=%s", path, status, duration, reqID)
	} else {
		log.Printf("route=%s status=%d duration=%s", path, status, duration)
	}
}

// NewChatCompletionsHandler creates a handler for production use.
func NewChatCompletionsHandler(cfg *config.Config) http.HandlerFunc {
	return newHandler(cfg, nil).ServeHTTP
}

// NewChatCompletionsHandlerWithClient creates a handler with custom HTTP client (for testing).
func NewChatCompletionsHandlerWithClient(cfg *config.Config, client *http.Client) http.HandlerFunc {
	return newHandler(cfg, client).ServeHTTP
}
