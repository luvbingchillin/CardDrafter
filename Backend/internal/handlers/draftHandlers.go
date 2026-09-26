package handlers

import (
	"backend/internal/repository"
	"encoding/json"
	"net/http"
)

func GetPacks(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	packsData := repository.GetPacks()
	resp := map[string]string{
		"message": "List of all packs",
		"data":    packsData,
	}
	json.NewEncoder(w).Encode(resp)
}
