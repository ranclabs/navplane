package orgsettings

import (
	"time"

	"github.com/google/uuid"
)

// ProviderSettings represents provider-level settings for an organization.
type ProviderSettings struct {
	ID            uuid.UUID `json:"id"`
	OrgID         uuid.UUID `json:"org_id"`
	Provider      string    `json:"provider"`
	Enabled       bool      `json:"enabled"`
	AllowedModels []string  `json:"allowed_models,omitempty"`
	BlockedModels []string  `json:"blocked_models,omitempty"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
}

// IsModelAllowed checks if a specific model is allowed for this provider.
// Rules:
// 1. If provider is disabled, no models are allowed
// 2. If blocked_models contains the model, it's blocked
// 3. If allowed_models is non-empty and doesn't contain the model, it's blocked
// 4. Otherwise, the model is allowed
func (ps *ProviderSettings) IsModelAllowed(model string) bool {
	if !ps.Enabled {
		return false
	}

	// Check blocked list first
	for _, blocked := range ps.BlockedModels {
		if blocked == model {
			return false
		}
	}

	// If allowed list is specified, model must be in it
	if len(ps.AllowedModels) > 0 {
		for _, allowed := range ps.AllowedModels {
			if allowed == model {
				return true
			}
		}
		return false
	}

	return true
}
