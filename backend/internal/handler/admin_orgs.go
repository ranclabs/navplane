package handler

import (
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"strconv"

	"lectr/internal/org"

	"github.com/google/uuid"
)

// AdminOrgsHandler handles admin operations for organizations.
type AdminOrgsHandler struct {
	manager *org.Manager
}

// NewAdminOrgsHandler creates a new admin orgs handler.
func NewAdminOrgsHandler(manager *org.Manager) *AdminOrgsHandler {
	return &AdminOrgsHandler{manager: manager}
}

// orgResponse is the JSON response for an organization.
// API key hash is never exposed.
type orgResponse struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	Enabled   bool   `json:"enabled"`
	CreatedAt string `json:"created_at"`
	UpdatedAt string `json:"updated_at"`
}

// createOrgResponse includes the API key (only on creation).
type createOrgResponse struct {
	orgResponse
	APIKey string `json:"api_key"`
}

func toOrgResponse(o *org.Org) orgResponse {
	return orgResponse{
		ID:        o.ID.String(),
		Name:      o.Name,
		Enabled:   o.Enabled,
		CreatedAt: o.CreatedAt.Format("2006-01-02T15:04:05Z"),
		UpdatedAt: o.UpdatedAt.Format("2006-01-02T15:04:05Z"),
	}
}

// List handles GET /admin/orgs
func (h *AdminOrgsHandler) List(w http.ResponseWriter, r *http.Request) {
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	offset, _ := strconv.Atoi(r.URL.Query().Get("offset"))

	orgs, err := h.manager.List(r.Context(), limit, offset)
	if err != nil {
		log.Printf("failed to list organizations: %v", err)
		writeAdminError(w, http.StatusInternalServerError, "failed to list organizations")
		return
	}

	response := make([]orgResponse, len(orgs))
	for i, o := range orgs {
		response[i] = toOrgResponse(o)
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"organizations": response,
		"count":         len(response),
	})
}

// Get handles GET /admin/orgs/{id}
func (h *AdminOrgsHandler) Get(w http.ResponseWriter, r *http.Request) {
	id, err := parseOrgID(r)
	if err != nil {
		writeAdminError(w, http.StatusBadRequest, "invalid organization ID")
		return
	}

	o, err := h.manager.GetByID(r.Context(), id)
	if err != nil {
		if errors.Is(err, org.ErrNotFound) {
			writeAdminError(w, http.StatusNotFound, "organization not found")
			return
		}
		log.Printf("failed to get organization: %v", err)
		writeAdminError(w, http.StatusInternalServerError, "failed to get organization")
		return
	}

	writeJSON(w, http.StatusOK, toOrgResponse(o))
}

// createOrgRequest is the JSON request for creating an organization.
type createOrgRequest struct {
	Name string `json:"name"`
}

// Create handles POST /admin/orgs
func (h *AdminOrgsHandler) Create(w http.ResponseWriter, r *http.Request) {
	var req createOrgRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeAdminError(w, http.StatusBadRequest, "invalid JSON")
		return
	}

	result, err := h.manager.Create(r.Context(), req.Name)
	if err != nil {
		if errors.Is(err, org.ErrInvalidName) {
			writeAdminError(w, http.StatusBadRequest, "name is required")
			return
		}
		log.Printf("failed to create organization: %v", err)
		writeAdminError(w, http.StatusInternalServerError, "failed to create organization")
		return
	}

	writeJSON(w, http.StatusCreated, createOrgResponse{
		orgResponse: toOrgResponse(result.Org),
		APIKey:      result.APIKey.Plaintext,
	})
}

// updateOrgRequest is the JSON request for updating an organization.
type updateOrgRequest struct {
	Name string `json:"name"`
}

// Update handles PUT /admin/orgs/{id}
func (h *AdminOrgsHandler) Update(w http.ResponseWriter, r *http.Request) {
	id, err := parseOrgID(r)
	if err != nil {
		writeAdminError(w, http.StatusBadRequest, "invalid organization ID")
		return
	}

	var req updateOrgRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeAdminError(w, http.StatusBadRequest, "invalid JSON")
		return
	}

	if err := h.manager.Update(r.Context(), id, req.Name); err != nil {
		if errors.Is(err, org.ErrNotFound) {
			writeAdminError(w, http.StatusNotFound, "organization not found")
			return
		}
		if errors.Is(err, org.ErrInvalidName) {
			writeAdminError(w, http.StatusBadRequest, "name is required")
			return
		}
		log.Printf("failed to update organization: %v", err)
		writeAdminError(w, http.StatusInternalServerError, "failed to update organization")
		return
	}

	o, err := h.manager.GetByID(r.Context(), id)
	if err != nil {
		log.Printf("failed to get updated organization: %v", err)
		writeAdminError(w, http.StatusInternalServerError, "failed to get organization")
		return
	}

	writeJSON(w, http.StatusOK, toOrgResponse(o))
}

// Delete handles DELETE /admin/orgs/{id}
func (h *AdminOrgsHandler) Delete(w http.ResponseWriter, r *http.Request) {
	id, err := parseOrgID(r)
	if err != nil {
		writeAdminError(w, http.StatusBadRequest, "invalid organization ID")
		return
	}

	if err := h.manager.Delete(r.Context(), id); err != nil {
		if errors.Is(err, org.ErrNotFound) {
			writeAdminError(w, http.StatusNotFound, "organization not found")
			return
		}
		log.Printf("failed to delete organization: %v", err)
		writeAdminError(w, http.StatusInternalServerError, "failed to delete organization")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// setEnabledRequest is the JSON request for enabling/disabling an org.
type setEnabledRequest struct {
	Enabled bool `json:"enabled"`
}

// SetEnabled handles PUT /admin/orgs/{id}/enabled
func (h *AdminOrgsHandler) SetEnabled(w http.ResponseWriter, r *http.Request) {
	id, err := parseOrgID(r)
	if err != nil {
		writeAdminError(w, http.StatusBadRequest, "invalid organization ID")
		return
	}

	var req setEnabledRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeAdminError(w, http.StatusBadRequest, "invalid JSON")
		return
	}

	if req.Enabled {
		err = h.manager.Enable(r.Context(), id)
	} else {
		err = h.manager.Disable(r.Context(), id)
	}

	if err != nil {
		if errors.Is(err, org.ErrNotFound) {
			writeAdminError(w, http.StatusNotFound, "organization not found")
			return
		}
		log.Printf("failed to set organization enabled state: %v", err)
		writeAdminError(w, http.StatusInternalServerError, "failed to update organization")
		return
	}

	o, err := h.manager.GetByID(r.Context(), id)
	if err != nil {
		log.Printf("failed to get updated organization: %v", err)
		writeAdminError(w, http.StatusInternalServerError, "failed to get organization")
		return
	}

	writeJSON(w, http.StatusOK, toOrgResponse(o))
}

// RotateAPIKey handles POST /admin/orgs/{id}/rotate-key
func (h *AdminOrgsHandler) RotateAPIKey(w http.ResponseWriter, r *http.Request) {
	id, err := parseOrgID(r)
	if err != nil {
		writeAdminError(w, http.StatusBadRequest, "invalid organization ID")
		return
	}

	newKey, err := h.manager.RotateAPIKey(r.Context(), id)
	if err != nil {
		if errors.Is(err, org.ErrNotFound) {
			writeAdminError(w, http.StatusNotFound, "organization not found")
			return
		}
		log.Printf("failed to rotate API key: %v", err)
		writeAdminError(w, http.StatusInternalServerError, "failed to rotate API key")
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{
		"api_key": newKey.Plaintext,
	})
}

// parseOrgID extracts the organization ID from the URL path.
func parseOrgID(r *http.Request) (uuid.UUID, error) {
	idStr := r.PathValue("id")
	if idStr == "" {
		return uuid.Nil, errors.New("missing organization ID")
	}
	return uuid.Parse(idStr)
}

// writeAdminError writes a JSON error response.
func writeAdminError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, map[string]any{
		"error": map[string]string{
			"message": message,
		},
	})
}

// writeJSON writes a JSON response with the given status code.
func writeJSON(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(data); err != nil {
		log.Printf("failed to encode JSON response: %v", err)
	}
}
