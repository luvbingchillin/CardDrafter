package handlers

import (
	"backend/internal/models"
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
)

// maxRequestBodyBytes caps incoming JSON request bodies at 1 MB to prevent memory exhaustion DoS.
const maxRequestBodyBytes = 1 << 20

// writeJSONError sends a consistent JSON {"error": "..."} payload with the given HTTP status code.
func writeJSONError(w http.ResponseWriter, message string, status int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(models.ErrorResponse{Error: message})
}

// AnalyticsProvider defines the analytics query contract required by AnalyticsHandler.
type AnalyticsProvider interface {
	GetAnalyticsBySet(ctx context.Context, setCode string) (*models.SetAnalytics, error)
}

// AnalyticsHandler handles HTTP requests for pack opening analytics and statistics.
type AnalyticsHandler struct {
	analyticsRepo AnalyticsProvider
}

// NewAnalyticsHandler constructs an AnalyticsHandler instance using an interface.
func NewAnalyticsHandler(analyticsRepo AnalyticsProvider) *AnalyticsHandler {
	return &AnalyticsHandler{
		analyticsRepo: analyticsRepo,
	}
}

// GetSetAnalytics handles GET /api/packs/{setCode}/analytics.
// Returns lifetime pull frequencies and total packs opened for the requested set.
func (h *AnalyticsHandler) GetSetAnalytics(w http.ResponseWriter, r *http.Request) {
	setCode := r.PathValue("setCode")
	if setCode == "" {
		writeJSONError(w, "set code is required", http.StatusBadRequest)
		return
	}

	analytics, err := h.analyticsRepo.GetAnalyticsBySet(r.Context(), setCode)
	if err != nil {
		slog.Error("Failed to retrieve set analytics", "set_code", setCode, "err", err)
		writeJSONError(w, "failed to retrieve analytics", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(analytics); err != nil {
		slog.Error("Failed to encode analytics response", "set_code", setCode, "err", err)
	}
}
