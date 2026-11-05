package handlers

import (
	"backend/internal/models"
	"backend/internal/utils"
	"context"
	"encoding/json"
	"net/http"
)

// PackOpener defines the booster pack generation contract required by PackHandler.
type PackOpener interface {
	OpenPack(ctx context.Context, setCode string, userID string) ([]*models.Card, error)
}

// PackSimulator defines the simulation service contract required by PackHandler.
type PackSimulator interface {
	RunSimulation(ctx context.Context, setCode string, totalPacks int) (*models.SimulationResult, error)
}

// PackHandler handles incoming HTTP requests related to booster pack generation and simulation.
// Decoupled completely from concrete service implementations via consumer interfaces.
type PackHandler struct {
	packService PackOpener
	simService  PackSimulator
}

// NewPackHandler constructs a PackHandler instance using consumer-driven interfaces.
func NewPackHandler(packService PackOpener, simService PackSimulator) *PackHandler {
	return &PackHandler{
		packService: packService,
		simService:  simService,
	}
}

// OpenPack handles POST /api/packs/{setCode}/open.
// It resolves optional user identity from cookies and generates a booster pack.
func (h *PackHandler) OpenPack(w http.ResponseWriter, r *http.Request) {
	setCode := r.PathValue("setCode")
	if setCode == "" {
		http.Error(w, "set code is required", http.StatusBadRequest)
		return
	}

	// Extract logged-in user if token cookie is present
	userID := "anonymous"
	if cookie, err := r.Cookie("token"); err == nil && cookie != nil {
		if claims, err := utils.ValidateToken(cookie.Value); err == nil {
			userID = claims.UserID
		}
	}

	cards, err := h.packService.OpenPack(r.Context(), setCode, userID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(cards)
}

// SimulatePacks handles POST /api/packs/{setCode}/simulate.
// It reads setCode strictly from the path and packCount from the JSON body.
func (h *PackHandler) SimulatePacks(w http.ResponseWriter, r *http.Request) {
	setCode := r.PathValue("setCode")
	if setCode == "" {
		http.Error(w, "set code is required", http.StatusBadRequest)
		return
	}

	var req models.SimulationRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(models.ErrorResponse{Error: "invalid request payload"})
		return
	}

	if req.PackCount <= 0 || req.PackCount > 100000 {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(models.ErrorResponse{Error: "packCount must be between 1 and 100,000"})
		return
	}

	result, err := h.simService.RunSimulation(r.Context(), setCode, req.PackCount)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(models.ErrorResponse{Error: err.Error()})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}
