package providerkey

import (
	"time"

	"github.com/google/uuid"
)

// ProviderKey represents a stored provider API key for an organization.
type ProviderKey struct {
	ID           uuid.UUID `json:"id"`
	OrgID        uuid.UUID `json:"org_id"`
	Provider     string    `json:"provider"`
	KeyAlias     string    `json:"key_alias"`
	EncryptedKey []byte    `json:"-"` // Never expose
	KeyNonce     []byte    `json:"-"` // Never expose
	EncryptedDEK []byte    `json:"-"` // Data Encryption Key, encrypted with KEK
	DEKNonce     []byte    `json:"-"` // Nonce for DEK encryption
	IsActive     bool      `json:"is_active"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

// ProviderKeyResponse is the safe response format (no secrets).
type ProviderKeyResponse struct {
	ID        uuid.UUID `json:"id"`
	OrgID     uuid.UUID `json:"org_id"`
	Provider  string    `json:"provider"`
	KeyAlias  string    `json:"key_alias"`
	IsActive  bool      `json:"is_active"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// ToResponse converts a ProviderKey to its safe response format.
func (pk *ProviderKey) ToResponse() ProviderKeyResponse {
	return ProviderKeyResponse{
		ID:        pk.ID,
		OrgID:     pk.OrgID,
		Provider:  pk.Provider,
		KeyAlias:  pk.KeyAlias,
		IsActive:  pk.IsActive,
		CreatedAt: pk.CreatedAt,
		UpdatedAt: pk.UpdatedAt,
	}
}
