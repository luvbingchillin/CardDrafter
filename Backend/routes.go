package main

import (
	"backend/internal/handlers"
	"net/http"
)

func setupRoutes() http.Handler {
	router := http.NewServeMux()
	router.HandleFunc("GET /api/packs", handlers.GetPacks)
	return router
}
