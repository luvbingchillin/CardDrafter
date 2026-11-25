package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"backend/internal/draft"
	"backend/internal/handlers"
	"backend/internal/repository"
	"backend/internal/services"

	"github.com/joho/godotenv"
)

func main() {
	if err := godotenv.Load(); err != nil {
		slog.Info("No .env file found; using system environment variables")
	}

	slog.Info("Starting Card Simulator Backend...")

	if os.Getenv("JWT_SECRET") == "" {
		slog.Warn("JWT_SECRET environment variable is not set; authenticated endpoints will reject tokens")
	}

	// 1. Initialize MongoDB connection
	mongoURI := os.Getenv("MONGODB_URI")
	if mongoURI == "" {
		slog.Error("MONGODB_URI environment variable is not set")
		os.Exit(1)
	}
	client, db, err := repository.ConnectDB(mongoURI, "cardsim_db")
	if err != nil {
		slog.Error("Database connection failed", "err", err)
		os.Exit(1)
	}
	defer func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = client.Disconnect(ctx)
	}()

	// 2. Initialize Redis client (caching & event streams)
	redisAddr := os.Getenv("REDIS_ADDR")
	if redisAddr == "" {
		redisAddr = "localhost:6379"
	}
	var eventStore draft.EventStore
	rdb, err := repository.NewRedisClient(redisAddr, "")
	if err != nil {
		slog.Warn("Redis unavailable; continuing without cache/stream persistence", "addr", redisAddr, "err", err)
	} else {
		defer func() { _ = rdb.Close() }()
		eventStore = repository.NewRedisEventStore(rdb)
	}

	userRepo := repository.NewUserRepository(db)
	cardRepo := repository.NewCardRepository(db)
	analyticsRepo := repository.NewAnalyticsRepository(db)

	authService := services.NewAuthService(userRepo)
	authHandler := handlers.NewAuthHandler(authService)
	packService := services.NewPackService(cardRepo, rdb)
	simService := services.NewSimulationService(cardRepo)
	packHandler := handlers.NewPackHandler(packService, simService)
	analyticsHandler := handlers.NewAnalyticsHandler(analyticsRepo)
	healthHandler := handlers.NewHealthHandler(client, rdb)
	draftHub := draft.NewHub(packService, eventStore)
	draftHandler := draft.NewDraftHandler(draftHub)

	router := setupRoutes(authHandler, packHandler, analyticsHandler, draftHandler, healthHandler)
	port := ":8000"

	// Production HTTP server with explicit Slowloris & connection timeouts
	srv := &http.Server{
		Addr:              port,
		Handler:           router,
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       15 * time.Second,
		WriteTimeout:      30 * time.Second,
		IdleTimeout:       60 * time.Second,
	}

	// Listen for SIGINT / SIGTERM for graceful container shutdown
	shutdownCtx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	go func() {
		slog.Info("Server listening", "url", "http://localhost"+port)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			slog.Error("HTTP server failed", "err", err)
			os.Exit(1)
		}
	}()

	<-shutdownCtx.Done()
	slog.Info("Shutdown signal received; draining in-flight connections...")

	graceCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := srv.Shutdown(graceCtx); err != nil {
		slog.Error("Graceful shutdown encountered error", "err", err)
	}
	slog.Info("Server stopped cleanly")
}
