package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"

	"backend/internal/handlers"
	"backend/internal/repository"
	"backend/internal/services"

	"github.com/joho/godotenv"
)

func main() {
	if err := godotenv.Load(); err != nil {
		log.Println("Note: No .env file found, using system environment variables")
	}

	fmt.Println("Starting Card Simulator Backend...")

	// 1. Initialize MongoDB connection
	mongoURI := os.Getenv("MONGODB_URI")
	if mongoURI == "" {
		log.Fatal("MONGODB_URI environment variable is not set!")
	}
	client, db, err := repository.ConnectDB(mongoURI, "cardsim_db")
	if err != nil {
		log.Fatalf("Database connection failed: %v", err)
	}
	defer client.Disconnect(context.Background())

	// 2. Initialize Redis client (caching & event streams)
	redisAddr := os.Getenv("REDIS_ADDR")
	if redisAddr == "" {
		redisAddr = "localhost:6379"
	}
	rdb, err := repository.NewRedisClient(redisAddr, "")
	if err != nil {
		log.Printf("Warning: Redis unavailable (%v). Continuing without cache/stream features.\n", err)
	} else {
		defer rdb.Close()
	}

	// 3. Initialize Analytics gRPC client
	analyticsAddr := os.Getenv("ANALYTICS_ADDR")
	if analyticsAddr == "" {
		analyticsAddr = "localhost:50051"
	}
	analyticsClient, err := services.NewAnalyticsClient(analyticsAddr)
	if err != nil {
		log.Printf("Warning: Analytics gRPC unavailable (%v). Continuing without live EV stats.\n", err)
	} else {
		defer analyticsClient.Close()
	}

	userRepo := repository.NewUserRepository(db)
	authService := services.NewAuthService(userRepo)
	authHandler := handlers.NewAuthHandler(authService)
	cardRepo := repository.NewCardRepository(db)
	packService := services.NewPackService(cardRepo, rdb)
	packHander := handlers.NewPackHandler(packService)

	router := setupRoutes(authHandler, packHander)
	port := ":8000"
	fmt.Println("Server listening on http://localhost" + port)
	if err := http.ListenAndServe(port, router); err != nil {
		log.Fatalf("Server failed to start: %v", err)
	}


}
