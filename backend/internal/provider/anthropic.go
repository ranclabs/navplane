package provider

import (
	"context"
	"fmt"
	"net/http"
	"time"
)

const (
	AnthropicName        = "anthropic"
	AnthropicDisplayName = "Anthropic"
	AnthropicBaseURL     = "https://api.anthropic.com/v1"
)

// Anthropic implements the Provider interface for Anthropic.
type Anthropic struct {
	client *http.Client
}

// NewAnthropic creates a new Anthropic provider.
func NewAnthropic() *Anthropic {
	return &Anthropic{
		client: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

func (a *Anthropic) Name() string {
	return AnthropicName
}

func (a *Anthropic) DisplayName() string {
	return AnthropicDisplayName
}

func (a *Anthropic) BaseURL() string {
	return AnthropicBaseURL
}

func (a *Anthropic) Models() []Model {
	return []Model{
		{
			ID:           "claude-3-5-sonnet-20241022",
			Name:         "Claude 3.5 Sonnet",
			Provider:     AnthropicName,
			ContextSize:  200000,
			Capabilities: []string{"chat", "vision"},
		},
		{
			ID:           "claude-3-5-haiku-20241022",
			Name:         "Claude 3.5 Haiku",
			Provider:     AnthropicName,
			ContextSize:  200000,
			Capabilities: []string{"chat", "vision"},
		},
		{
			ID:           "claude-3-opus-20240229",
			Name:         "Claude 3 Opus",
			Provider:     AnthropicName,
			ContextSize:  200000,
			Capabilities: []string{"chat", "vision"},
		},
	}
}

func (a *Anthropic) AuthHeader() string {
	return "x-api-key"
}

func (a *Anthropic) FormatAuthValue(apiKey string) string {
	return apiKey // Anthropic doesn't use Bearer prefix
}

// ValidateKey tests if an Anthropic API key is valid.
// Anthropic doesn't have a /models endpoint, so we use a minimal messages request.
func (a *Anthropic) ValidateKey(ctx context.Context, apiKey string) error {
	// For Anthropic, we'll make a request to check auth
	// The /messages endpoint with an empty body will return 400 but not 401 if key is valid
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, AnthropicBaseURL+"/models", nil)
	if err != nil {
		return fmt.Errorf("%w: %v", ErrKeyValidation, err)
	}

	req.Header.Set(a.AuthHeader(), a.FormatAuthValue(apiKey))
	req.Header.Set("anthropic-version", "2023-06-01")

	resp, err := a.client.Do(req)
	if err != nil {
		return fmt.Errorf("%w: %v", ErrKeyValidation, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusUnauthorized {
		return ErrInvalidAPIKey
	}

	// 200 = valid key, other errors might be rate limits etc. which still indicate valid key
	if resp.StatusCode == http.StatusOK || resp.StatusCode == http.StatusTooManyRequests {
		return nil
	}

	// 403 often means invalid key for Anthropic
	if resp.StatusCode == http.StatusForbidden {
		return ErrInvalidAPIKey
	}

	return nil
}
