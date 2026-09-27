package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"

	"backend/internal/repository"

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

	userRepo := repository.NewUserRepository(db)
	_ = userRepo // Ready for our auth service!

	router := setupRoutes()
	port := ":8000"
	fmt.Println("Server listening on http://localhost" + port)
	if err := http.ListenAndServe(port, router); err != nil {
		log.Fatalf("Server failed to start: %v", err)
	}
}
