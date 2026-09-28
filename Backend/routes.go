package main

import (
	"backend/internal/handlers"
	"net/http"
)

func setupRoutes(authHandler *handlers.AuthHandler) http.Handler {
	router := http.NewServeMux()
	router.HandleFunc("GET /api/packs", handlers.GetPacks)
	router.HandleFunc("POST /api/auth/login", authHandler.Login)
	router.HandleFunc("POST /api/auth/register", authHandler.Register)
	return router
}
