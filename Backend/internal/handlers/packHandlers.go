package handlers

import (
	"backend/internal/models"
	"backend/internal/utils"
	"context"
	"encoding/json"
	"log/slog"
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
		writeJSONError(w, "set code is required", http.StatusBadRequest)
		return
	}

	userID := "anonymous"
	if cookie, err := r.Cookie("token"); err == nil && cookie != nil {
		if claims, err := utils.ValidateToken(cookie.Value); err == nil {
			userID = claims.UserID
		}
	}

	cards, err := h.packService.OpenPack(r.Context(), setCode, userID)
	if err != nil {
		slog.Error("Failed to open booster pack", "set_code", setCode, "user_id", userID, "err", err)
		writeJSONError(w, "failed to generate booster pack for set "+setCode, http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(cards); err != nil {
		slog.Error("Failed to encode booster pack response", "set_code", setCode, "err", err)
	}
}

// SimulatePacks handles POST /api/packs/{setCode}/simulate.
// It reads setCode strictly from the path and packCount from the JSON body.
func (h *PackHandler) SimulatePacks(w http.ResponseWriter, r *http.Request) {
	setCode := r.PathValue("setCode")
	if setCode == "" {
		writeJSONError(w, "set code is required", http.StatusBadRequest)
		return
	}

	r.Body = http.MaxBytesReader(w, r.Body, maxRequestBodyBytes)
	var req models.SimulationRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSONError(w, "invalid request payload", http.StatusBadRequest)
		return
	}

	if req.PackCount <= 0 || req.PackCount > 100000 {
		writeJSONError(w, "packCount must be between 1 and 100,000", http.StatusBadRequest)
		return
	}

	result, err := h.simService.RunSimulation(r.Context(), setCode, req.PackCount)
	if err != nil {
		slog.Error("Failed to run pack simulation", "set_code", setCode, "pack_count", req.PackCount, "err", err)
		writeJSONError(w, "failed to execute pack simulation", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(result); err != nil {
		slog.Error("Failed to encode simulation response", "set_code", setCode, "err", err)
	}
}
