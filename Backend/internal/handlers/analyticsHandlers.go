package handlers

import (
	"backend/internal/repository"
	"encoding/json"
	"net/http"
)

// AnalyticsHandler handles HTTP requests for pack opening analytics and statistics.
type AnalyticsHandler struct {
	analyticsRepo *repository.AnalyticsRepository
}

// NewAnalyticsHandler constructs an AnalyticsHandler instance.
func NewAnalyticsHandler(analyticsRepo *repository.AnalyticsRepository) *AnalyticsHandler {
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
