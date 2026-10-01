package handlers

import (
	"backend/internal/models"
	"encoding/json"
	"net/http"
)

var availablePacks = []models.PackInfo{
	{SetCode: "BLB", Name: "Bloomburrow", Image: "/packs/BLB.png"},
	{SetCode: "FDN", Name: "Foundations", Image: "/packs/FDN.png"},
	{SetCode: "MKM", Name: "Murders at Karlov Manor", Image: "/packs/MKM.png"},
	{SetCode: "WOE", Name: "Wilds of Eldraine", Image: "/packs/WOE.png"},
	{SetCode: "LCI", Name: "The Lost Caverns of Ixalan", Image: "/packs/LCI.png"},
	{SetCode: "ONE", Name: "Phyrexia: All Will Be One", Image: "/packs/ONE.png"},
	{SetCode: "BRO", Name: "The Brothers' War", Image: "/packs/BRO.png"},
	{SetCode: "DMU", Name: "Dominaria United", Image: "/packs/DMU.png"},
	{SetCode: "NEO", Name: "Kamigawa: Neon Dynasty", Image: "/packs/NEO.png"},
}

func GetPacks(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(availablePacks); err != nil {
		http.Error(w, "Failed to encode pack data", http.StatusInternalServerError)
		return
	}
}
