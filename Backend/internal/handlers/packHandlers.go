package handlers

import (
	"backend/internal/services"
	"encoding/json"
	"net/http"
)

type PackHandler struct{
	packService *services.PackService
}

func NewPackHandler(packService *services.PackService) *PackHandler{
	return &PackHandler{packService: packService}
}

func (h *PackHandler) OpenPack(w http.ResponseWriter, r *http.Request){
	setCode := r.PathValue("setCode")
	if setCode == ""{
		http.Error(w, "set code is required", http.StatusBadRequest)
		return
	}
	cards, err:=h.packService.OpenPack(r.Context(), setCode)
	if err != nil{
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type","application/json")
	json.NewEncoder(w).Encode(cards)
}