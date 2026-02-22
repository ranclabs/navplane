package handler

import (
	"encoding/json"
	"log"
	"net/http"

	"lectr/internal/config"
)

// statusHandler returns an HTTP handler that has access to the config.
// This pattern allows handlers to access provider configuration without
// reading environment variables directly in request handling code.
//
// Note: Config is currently unused but available for future enhancements
// such as verifying provider connectivity or including environment info
// in the status response.
func statusHandler(cfg *config.Config) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(map[string]any{
			"service": "lectr",
			"version": "0.1.0",
			"status":  "operational",
		}); err != nil {
			log.Printf("failed to write status response: %v", err)
		}
	}
}
