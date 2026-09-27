package handlers

import (
	"encoding/json"
	"net/http"
)

func GetPacks(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	resp := map[string]string{
		"message": "List of all packs",
		"data":    "placeholder packs", // <-- Just use a placeholder for now!
	}
	json.NewEncoder(w).Encode(resp)
}
