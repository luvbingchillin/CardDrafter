package main

import (
	"backend/internal/handlers"
	"net/http"
)

func setupRoutes(authHandler *handlers.AuthHandler, packHandler *handlers.PackHandler) http.Handler {
	router := http.NewServeMux()
	router.HandleFunc("GET /api/packs", handlers.GetPacks)
	router.HandleFunc("POST /api/auth/login", authHandler.Login)
	router.HandleFunc("POST /api/auth/register", authHandler.Register)
	router.HandleFunc("GET /api/auth/google/callback", authHandler.HandleGoogleCallback)
	router.HandleFunc("GET /api/auth/google", authHandler.HandleGoogleLogin)
	router.HandleFunc("POST /api/packs/{setCode}/open", packHandler.OpenPack)
	return router
}
