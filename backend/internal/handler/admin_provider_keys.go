package handler

import (
	"encoding/json"
	"errors"
	"log"
	"net/http"

	"navplane/internal/provider"
	"navplane/internal/providerkey"

	"github.com/google/uuid"
)

// AdminProviderKeysHandler handles admin operations for provider keys.
type AdminProviderKeysHandler struct {
	manager  *providerkey.Manager
	registry *provider.Registry
}

// NewAdminProviderKeysHandler creates a new admin provider keys handler.
func NewAdminProviderKeysHandler(manager *providerkey.Manager, registry *provider.Registry) *AdminProviderKeysHandler {
	return &AdminProviderKeysHandler{
		manager:  manager,
		registry: registry,
	}
}

// List handles GET /admin/orgs/{id}/provider-keys
func (h *AdminProviderKeysHandler) List(w http.ResponseWriter, r *http.Request) {
	orgID, err := parseOrgID(r)
	if err != nil {
		writeAdminError(w, http.StatusBadRequest, "invalid organization ID")
		return
	}

	keys, err := h.manager.ListByOrg(r.Context(), orgID)
	if err != nil {
		log.Printf("failed to list provider keys: %v", err)
		writeAdminError(w, http.StatusInternalServerError, "failed to list provider keys")
		return
	}

	response := make([]providerkey.ProviderKeyResponse, len(keys))
	for i, k := range keys {
		response[i] = k.ToResponse()
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"provider_keys": response,
		"count":         len(response),
	})
}

// createProviderKeyRequest is the request body for creating a provider key.
type createProviderKeyRequest struct {
	Provider    string `json:"provider"`
	KeyAlias    string `json:"key_alias"`
	APIKey      string `json:"api_key"`
	ValidateKey bool   `json:"validate_key"`
}

// Create handles POST /admin/orgs/{id}/provider-keys
func (h *AdminProviderKeysHandler) Create(w http.ResponseWriter, r *http.Request) {
	orgID, err := parseOrgID(r)
	if err != nil {
		writeAdminError(w, http.StatusBadRequest, "invalid organization ID")
		return
	}

	var req createProviderKeyRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeAdminError(w, http.StatusBadRequest, "invalid JSON")
		return
	}

	pk, err := h.manager.Create(r.Context(), providerkey.CreateInput{
		OrgID:       orgID,
		Provider:    req.Provider,
		KeyAlias:    req.KeyAlias,
		APIKey:      req.APIKey,
		ValidateKey: req.ValidateKey,
	})

	if err != nil {
		switch {
		case errors.Is(err, providerkey.ErrInvalidProvider):
			writeAdminError(w, http.StatusBadRequest, "unsupported provider")
		case errors.Is(err, providerkey.ErrInvalidAlias):
			writeAdminError(w, http.StatusBadRequest, "key alias is required")
		case errors.Is(err, providerkey.ErrInvalidKey):
			writeAdminError(w, http.StatusBadRequest, "invalid API key")
		case errors.Is(err, providerkey.ErrKeyExists):
			writeAdminError(w, http.StatusConflict, "provider key already exists")
		default:
			log.Printf("failed to create provider key: %v", err)
			writeAdminError(w, http.StatusInternalServerError, "failed to create provider key")
		}
		return
	}

	writeJSON(w, http.StatusCreated, pk.ToResponse())
}

// Delete handles DELETE /admin/orgs/{id}/provider-keys/{keyId}
func (h *AdminProviderKeysHandler) Delete(w http.ResponseWriter, r *http.Request) {
	keyIDStr := r.PathValue("keyId")
	if keyIDStr == "" {
		writeAdminError(w, http.StatusBadRequest, "missing provider key ID")
		return
	}

	keyID, err := uuid.Parse(keyIDStr)
	if err != nil {
		writeAdminError(w, http.StatusBadRequest, "invalid provider key ID")
		return
	}

	if err := h.manager.Delete(r.Context(), keyID); err != nil {
		if errors.Is(err, providerkey.ErrNotFound) {
			writeAdminError(w, http.StatusNotFound, "provider key not found")
			return
		}
		log.Printf("failed to delete provider key: %v", err)
		writeAdminError(w, http.StatusInternalServerError, "failed to delete provider key")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// handleListProviders handles GET /providers
func handleListProviders(registry *provider.Registry) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		providers := registry.List()

		response := make([]map[string]any, len(providers))
		for i, p := range providers {
			response[i] = map[string]any{
				"name":         p.Name(),
				"display_name": p.DisplayName(),
				"models":       p.Models(),
			}
		}

		writeJSON(w, http.StatusOK, map[string]any{
			"providers": response,
		})
	}
}
