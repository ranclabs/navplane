package provider

import (
	"context"
	"fmt"
	"net/http"
	"time"
)

const (
	OpenAIName        = "openai"
	OpenAIDisplayName = "OpenAI"
	OpenAIBaseURL     = "https://api.openai.com/v1"
)

// OpenAI implements the Provider interface for OpenAI.
type OpenAI struct {
	client *http.Client
}

// NewOpenAI creates a new OpenAI provider.
func NewOpenAI() *OpenAI {
	return &OpenAI{
		client: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

func (o *OpenAI) Name() string {
	return OpenAIName
}

func (o *OpenAI) DisplayName() string {
	return OpenAIDisplayName
}

func (o *OpenAI) BaseURL() string {
	return OpenAIBaseURL
}

func (o *OpenAI) Models() []Model {
	return []Model{
		{
			ID:           "gpt-4o",
			Name:         "GPT-4o",
			Provider:     OpenAIName,
			ContextSize:  128000,
			Capabilities: []string{"chat", "vision", "function_calling"},
		},
		{
			ID:           "gpt-4o-mini",
			Name:         "GPT-4o Mini",
			Provider:     OpenAIName,
			ContextSize:  128000,
			Capabilities: []string{"chat", "vision", "function_calling"},
		},
		{
			ID:           "gpt-4-turbo",
			Name:         "GPT-4 Turbo",
			Provider:     OpenAIName,
			ContextSize:  128000,
			Capabilities: []string{"chat", "vision", "function_calling"},
		},
		{
			ID:           "o1",
			Name:         "O1",
			Provider:     OpenAIName,
			ContextSize:  200000,
			Capabilities: []string{"chat", "reasoning"},
		},
		{
			ID:           "o1-mini",
			Name:         "O1 Mini",
			Provider:     OpenAIName,
			ContextSize:  128000,
			Capabilities: []string{"chat", "reasoning"},
		},
	}
}

func (o *OpenAI) AuthHeader() string {
	return "Authorization"
}

func (o *OpenAI) FormatAuthValue(apiKey string) string {
	return "Bearer " + apiKey
}

// ValidateKey tests if an OpenAI API key is valid by calling the /models endpoint.
func (o *OpenAI) ValidateKey(ctx context.Context, apiKey string) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, OpenAIBaseURL+"/models", nil)
	if err != nil {
		return fmt.Errorf("%w: %v", ErrKeyValidation, err)
	}

	req.Header.Set(o.AuthHeader(), o.FormatAuthValue(apiKey))

	resp, err := o.client.Do(req)
	if err != nil {
		return fmt.Errorf("%w: %v", ErrKeyValidation, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusUnauthorized {
		return ErrInvalidAPIKey
	}

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("%w: unexpected status %d", ErrKeyValidation, resp.StatusCode)
	}

	return nil
}
