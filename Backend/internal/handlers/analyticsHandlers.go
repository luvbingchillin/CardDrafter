package handlers

import (
	"backend/internal/models"
	"context"
	"encoding/json"
	"net/http"
)

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
		http.Error(w, "set code is required", http.StatusBadRequest)
		return
	}

	analytics, err := h.analyticsRepo.GetAnalyticsBySet(r.Context(), setCode)
	if err != nil {
		http.Error(w, "failed to retrieve analytics: "+err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(analytics); err != nil {
		http.Error(w, "failed to encode response", http.StatusInternalServerError)
	}
}
