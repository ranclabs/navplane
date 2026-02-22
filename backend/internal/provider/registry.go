package provider

import "sync"

// Registry holds all registered providers.
type Registry struct {
	mu        sync.RWMutex
	providers map[string]Provider
}

// NewRegistry creates a new provider registry with default providers.
func NewRegistry() *Registry {
	r := &Registry{
		providers: make(map[string]Provider),
	}

	// Register default providers
	r.Register(NewOpenAI())
	r.Register(NewAnthropic())

	return r
}

// Register adds a provider to the registry.
func (r *Registry) Register(p Provider) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.providers[p.Name()] = p
}

// Get returns a provider by name.
func (r *Registry) Get(name string) (Provider, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	p, ok := r.providers[name]
	if !ok {
		return nil, ErrProviderNotFound
	}
	return p, nil
}

// List returns all registered providers.
func (r *Registry) List() []Provider {
	r.mu.RLock()
	defer r.mu.RUnlock()

	providers := make([]Provider, 0, len(r.providers))
	for _, p := range r.providers {
		providers = append(providers, p)
	}
	return providers
}

// ListModels returns all models from all providers.
func (r *Registry) ListModels() []Model {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var models []Model
	for _, p := range r.providers {
		models = append(models, p.Models()...)
	}
	return models
}

// Names returns the names of all registered providers.
func (r *Registry) Names() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	names := make([]string, 0, len(r.providers))
	for name := range r.providers {
		names = append(names, name)
	}
	return names
}
