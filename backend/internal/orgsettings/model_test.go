package orgsettings

import (
	"testing"
)

func TestProviderSettings_IsModelAllowed(t *testing.T) {
	tests := []struct {
		name          string
		settings      ProviderSettings
		model         string
		expected      bool
	}{
		{
			name:     "disabled provider blocks all",
			settings: ProviderSettings{Enabled: false},
			model:    "gpt-4",
			expected: false,
		},
		{
			name:     "enabled provider with no restrictions allows all",
			settings: ProviderSettings{Enabled: true},
			model:    "gpt-4",
			expected: true,
		},
		{
			name: "blocked model is blocked",
			settings: ProviderSettings{
				Enabled:       true,
				BlockedModels: []string{"gpt-4"},
			},
			model:    "gpt-4",
			expected: false,
		},
		{
			name: "non-blocked model is allowed",
			settings: ProviderSettings{
				Enabled:       true,
				BlockedModels: []string{"gpt-4"},
			},
			model:    "gpt-3.5-turbo",
			expected: true,
		},
		{
			name: "allowed list restricts to only those models",
			settings: ProviderSettings{
				Enabled:       true,
				AllowedModels: []string{"gpt-4", "gpt-4-turbo"},
			},
			model:    "gpt-4",
			expected: true,
		},
		{
			name: "model not in allowed list is blocked",
			settings: ProviderSettings{
				Enabled:       true,
				AllowedModels: []string{"gpt-4", "gpt-4-turbo"},
			},
			model:    "gpt-3.5-turbo",
			expected: false,
		},
		{
			name: "blocked takes precedence over allowed",
			settings: ProviderSettings{
				Enabled:       true,
				AllowedModels: []string{"gpt-4"},
				BlockedModels: []string{"gpt-4"},
			},
			model:    "gpt-4",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.settings.IsModelAllowed(tt.model)
			if result != tt.expected {
				t.Errorf("IsModelAllowed(%q) = %v, want %v", tt.model, result, tt.expected)
			}
		})
	}
}
