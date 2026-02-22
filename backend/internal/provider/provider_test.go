package provider

import (
	"testing"
)

func TestOpenAI_ImplementsProvider(t *testing.T) {
	var _ Provider = (*OpenAI)(nil)
}

func TestAnthropic_ImplementsProvider(t *testing.T) {
	var _ Provider = (*Anthropic)(nil)
}

func TestOpenAI_Properties(t *testing.T) {
	p := NewOpenAI()

	if p.Name() != "openai" {
		t.Errorf("expected name 'openai', got %q", p.Name())
	}

	if p.DisplayName() != "OpenAI" {
		t.Errorf("expected display name 'OpenAI', got %q", p.DisplayName())
	}

	if p.BaseURL() != "https://api.openai.com/v1" {
		t.Errorf("expected base URL 'https://api.openai.com/v1', got %q", p.BaseURL())
	}

	if p.AuthHeader() != "Authorization" {
		t.Errorf("expected auth header 'Authorization', got %q", p.AuthHeader())
	}

	if p.FormatAuthValue("sk-test") != "Bearer sk-test" {
		t.Errorf("expected 'Bearer sk-test', got %q", p.FormatAuthValue("sk-test"))
	}
}

func TestAnthropic_Properties(t *testing.T) {
	p := NewAnthropic()

	if p.Name() != "anthropic" {
		t.Errorf("expected name 'anthropic', got %q", p.Name())
	}

	if p.DisplayName() != "Anthropic" {
		t.Errorf("expected display name 'Anthropic', got %q", p.DisplayName())
	}

	if p.BaseURL() != "https://api.anthropic.com/v1" {
		t.Errorf("expected base URL 'https://api.anthropic.com/v1', got %q", p.BaseURL())
	}

	if p.AuthHeader() != "x-api-key" {
		t.Errorf("expected auth header 'x-api-key', got %q", p.AuthHeader())
	}

	if p.FormatAuthValue("sk-ant-test") != "sk-ant-test" {
		t.Errorf("expected 'sk-ant-test', got %q", p.FormatAuthValue("sk-ant-test"))
	}
}

func TestOpenAI_Models(t *testing.T) {
	p := NewOpenAI()
	models := p.Models()

	if len(models) == 0 {
		t.Error("expected at least one model")
	}

	// Check that gpt-4o is in the list
	found := false
	for _, m := range models {
		if m.ID == "gpt-4o" {
			found = true
			if m.Provider != "openai" {
				t.Errorf("expected provider 'openai', got %q", m.Provider)
			}
			break
		}
	}
	if !found {
		t.Error("expected to find gpt-4o model")
	}
}

func TestAnthropic_Models(t *testing.T) {
	p := NewAnthropic()
	models := p.Models()

	if len(models) == 0 {
		t.Error("expected at least one model")
	}

	// Check that claude-3-5-sonnet is in the list
	found := false
	for _, m := range models {
		if m.ID == "claude-3-5-sonnet-20241022" {
			found = true
			if m.Provider != "anthropic" {
				t.Errorf("expected provider 'anthropic', got %q", m.Provider)
			}
			break
		}
	}
	if !found {
		t.Error("expected to find claude-3-5-sonnet model")
	}
}

func TestRegistry_Get(t *testing.T) {
	r := NewRegistry()

	p, err := r.Get("openai")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if p.Name() != "openai" {
		t.Errorf("expected openai provider, got %q", p.Name())
	}

	p, err = r.Get("anthropic")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if p.Name() != "anthropic" {
		t.Errorf("expected anthropic provider, got %q", p.Name())
	}
}

func TestRegistry_GetNotFound(t *testing.T) {
	r := NewRegistry()

	_, err := r.Get("unknown")
	if err != ErrProviderNotFound {
		t.Errorf("expected ErrProviderNotFound, got %v", err)
	}
}

func TestRegistry_List(t *testing.T) {
	r := NewRegistry()

	providers := r.List()
	if len(providers) != 2 {
		t.Errorf("expected 2 providers, got %d", len(providers))
	}
}

func TestRegistry_ListModels(t *testing.T) {
	r := NewRegistry()

	models := r.ListModels()
	if len(models) == 0 {
		t.Error("expected at least one model")
	}

	// Should have models from both providers
	hasOpenAI := false
	hasAnthropic := false
	for _, m := range models {
		if m.Provider == "openai" {
			hasOpenAI = true
		}
		if m.Provider == "anthropic" {
			hasAnthropic = true
		}
	}

	if !hasOpenAI {
		t.Error("expected models from openai")
	}
	if !hasAnthropic {
		t.Error("expected models from anthropic")
	}
}

func TestRegistry_Names(t *testing.T) {
	r := NewRegistry()

	names := r.Names()
	if len(names) != 2 {
		t.Errorf("expected 2 names, got %d", len(names))
	}

	hasOpenAI := false
	hasAnthropic := false
	for _, name := range names {
		if name == "openai" {
			hasOpenAI = true
		}
		if name == "anthropic" {
			hasAnthropic = true
		}
	}

	if !hasOpenAI || !hasAnthropic {
		t.Errorf("expected openai and anthropic, got %v", names)
	}
}
