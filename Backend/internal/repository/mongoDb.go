package repository

import (
	"context"
	"fmt"
	"log"
	"time"

	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

// ConnectDB establishes a connection to MongoDB Atlas and returns the client and database
func ConnectDB(uri string, dbName string) (*mongo.Client, *mongo.Database, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Configure client with Server API version 1 (recommended for Atlas)
	serverAPI := options.ServerAPI(options.ServerAPIVersion1)
	clientOpts := options.Client().ApplyURI(uri).SetServerAPIOptions(serverAPI)

	client, err := mongo.Connect(clientOpts)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create client: %w", err)
	}

	// Verify the connection with a ping
	if err := client.Ping(ctx, nil); err != nil {
		return nil, nil, fmt.Errorf("failed to ping Atlas cluster: %w", err)
	}

	log.Println(" Connected to MongoDB Atlas successfully!")
	db := client.Database(dbName)
	return client, db, nil
}
