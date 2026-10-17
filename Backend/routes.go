package main

import (
	"backend/internal/draft"
	"backend/internal/handlers"
	"backend/internal/middleware"
	"net/http"
)

func setupRoutes(
	authHandler *handlers.AuthHandler,
	packHandler *handlers.PackHandler,
	analyticsHandler *handlers.AnalyticsHandler,
	draftHandler *draft.DraftHandler,
) http.Handler {
	router := http.NewServeMux()
	router.HandleFunc("GET /api/packs", handlers.GetPacks)
	router.HandleFunc("POST /api/auth/login", authHandler.Login)
	router.HandleFunc("POST /api/auth/register", authHandler.Register)
	router.HandleFunc("POST /api/auth/logout", authHandler.Logout)
	router.HandleFunc("GET /api/auth/google/callback", authHandler.HandleGoogleCallback)
	router.HandleFunc("GET /api/auth/google", authHandler.HandleGoogleLogin)
	router.Handle("GET /api/auth/me", middleware.AuthMiddleware(http.HandlerFunc(authHandler.GetMe)))
	router.HandleFunc("POST /api/packs/{setCode}/open", packHandler.OpenPack)
	router.HandleFunc("GET /api/packs/{setCode}/analytics", analyticsHandler.GetSetAnalytics)
	router.Handle("GET /api/draft/{roomId}/ws", middleware.AuthMiddleware(http.HandlerFunc(draftHandler.ServeWs)))
	return router
}
