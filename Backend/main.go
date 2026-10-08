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

	"github.com/joho/godotenv" // <-- import godotenv
)

func main() {
	// Load .env file at application startup
	if err := godotenv.Load(); err != nil {
		log.Println("Note: No .env file found, reading from system environment")
	}

	fmt.Println("Starting Card Simulator Backend...")

	mongoURI := os.Getenv("MONGODB_URI")
	if mongoURI == "" {
		log.Fatal("MONGODB_URI environment variable is not set!")
	}

	// Connect to Atlas
	client, db, err := repository.ConnectDB(mongoURI, "cardsim_db")
	if err != nil {
		log.Fatalf("Database connection failed: %v", err)
	}
	defer client.Disconnect(context.Background())


	redisAddr := os.Getenv("REDIS_ADDR")
	if redisAddr == "" {
		redisAddr = "localhost:6379"
	}
	rdb, err := repository.NewRedisClient(redisAddr, "")
	if err != nil {
		log.Printf(" Warning: Redis is unavailable (%v). Continuing without cache/draft features.\n", err)
	} else {
		defer rdb.Close()

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
