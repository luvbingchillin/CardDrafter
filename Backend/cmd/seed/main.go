package main

import (
	"backend/internal/models"
	"backend/internal/repository"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"

	"github.com/joho/godotenv"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
)

func main() {
	fmt.Println("\n========================================")
	fmt.Println("       MTG Card Seeder & Importer       ")
	fmt.Println("========================================")

	// =========================================================================
	// SECTION 1: CLI ARGUMENTS & FRIENDLY HELP
	// =========================================================================
	// If the user didn't provide a set code, display a friendly help screen
	// with popular draft sets instead of abruptly crashing.
	if len(os.Args) < 2 {
		fmt.Println("\n👋 Welcome! To seed a set into your database, pass a set code:")
		fmt.Println("   go run ./cmd/seed <SET_CODE>")
		fmt.Println()
		fmt.Println("Popular draft sets you can try:")
		fmt.Println("   • BLB  - Bloomburrow")
		fmt.Println("   • DSK  - Duskmourn: House of Horror")
		fmt.Println("   • MH3  - Modern Horizons 3")
		fmt.Println("   • OTJ  - Outlaws of Thunder Junction")
		fmt.Println("   • MKM  - Murders at Karlov Manor")
		fmt.Println()
		fmt.Println("Example:")
		fmt.Println("   go run ./cmd/seed BLB")
		fmt.Println()
		os.Exit(0)
	}

	// Clean user input: strip accidental whitespace and enforce uppercase (e.g. "blb" -> "BLB")
	setCode := strings.ToUpper(strings.TrimSpace(os.Args[1]))
	fmt.Printf("\n🎯 Target Set: %s\n", setCode)

	// =========================================================================
	// SECTION 2: CONFIGURATION & DATABASE CONNECTION
	// =========================================================================
	fmt.Println("⏳ [1/4] Connecting to MongoDB Atlas...")
	if err := godotenv.Load(); err != nil {
		log.Println("   Note: .env file not found, falling back to system environment variables")
	}

	mongoURI := os.Getenv("MONGODB_URI")
	if mongoURI == "" {
		log.Fatal("❌ Error: MONGODB_URI is not set in environment or .env file")
	}

	client, db, err := repository.ConnectDB(mongoURI, "cardsim_db")
	if err != nil {
		log.Fatalf("❌ Database connection failed: %v", err)
	}
	defer client.Disconnect(context.Background())

	// =========================================================================
	// SECTION 3: HTTP DOWNLOAD FROM MTGJSON
	// =========================================================================
	url := fmt.Sprintf("https://mtgjson.com/api/v5/%s.json", setCode)
	fmt.Printf("⏳ [2/4] Downloading set data from MTGJSON (%s)...\n", url)

	resp, err := http.Get(url)
	if err != nil {
		log.Fatalf("❌ Failed to download set data: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		log.Fatalf("❌ MTGJSON returned HTTP %d. Please verify that '%s' is a valid set code.", resp.StatusCode, setCode)
	}

	// =========================================================================
	// SECTION 4: MTGJSON DATA TRANSFER OBJECTS (DTOs)
	// =========================================================================
	// These structs mirror the third-party MTGJSON schema. We keep them local to
	// this function to prevent coupling our internal models with external API schemas.
	type mtgjsonCard struct {
		UUID        string   `json:"uuid"`
		Name        string   `json:"name"`
		ManaCost    string   `json:"manaCost"`
		ManaValue   float64  `json:"manaValue"`
		Rarity      string   `json:"rarity"`
		Types       []string `json:"types"`
		Subtypes    []string `json:"subtypes"`
		Text        string   `json:"text"`
		Power       string   `json:"power"`
		Toughness   string   `json:"toughness"`
		Number      string   `json:"number"`
		Colors      []string `json:"colors"`
		Identifiers struct {
			ScryfallID string `json:"scryfallId"` // Required to build Scryfall image URLs
		} `json:"identifiers"`
	}

	type mtgjsonResponse struct {
		Data struct {
			Code  string        `json:"code"`
			Name  string        `json:"name"`
			Cards []mtgjsonCard `json:"cards"`
		} `json:"data"`
	}

	// =========================================================================
	// SECTION 5: STREAMING JSON PARSING
	// =========================================================================
	// json.NewDecoder reads directly from the network response stream (resp.Body).
	// This avoids buffering the entire multi-megabyte JSON file into RAM before decoding.
	var payload mtgjsonResponse
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		log.Fatalf("Failed to parse MTGJSON payload: %v", err)
	}
	fmt.Printf("Downloaded set '%s' containing %d total cards\n", payload.Data.Name, len(payload.Data.Cards))

	cardCollection := db.Collection("cards")
	ctx := context.Background()

	// =========================================================================
	// SECTION 6: IDEMPOTENT CLEANUP (PREVENT DUPLICATES)
	// =========================================================================
	// If this script is run multiple times for the same set, wipe the existing cards
	// for this set code first so we never end up with duplicate card records.
	// bson.M is an unordered map representing a MongoDB query filter {"set_code": setCode}
	delResult, err := cardCollection.DeleteMany(ctx, bson.M{"set_code": setCode})
	if err != nil {
		log.Fatalf("Failed to clear old cards: %v", err)
	}
	if delResult.DeletedCount > 0 {
		fmt.Printf("Cleared %d existing cards for set %s\n", delResult.DeletedCount, setCode)
	}

	// =========================================================================
	// SECTION 7: DATA MAPPING TO INTERNAL DOMAIN MODEL
	// =========================================================================
	// Transform external MTGJSON cards into our own models.Card MongoDB documents.
	// We convert them into a slice of []any because the mongo-driver's InsertMany
	// takes a slice of empty interfaces ([]any / []interface{}).
	var docs []any
	for _, c := range payload.Data.Cards {
		// Filter out cards without a Scryfall ID (e.g., promotional variants/tokens)
		if c.Identifiers.ScryfallID == "" {
			continue
		}

		doc := models.Card{
			UUID:            c.UUID,
			ScryfallID:      c.Identifiers.ScryfallID,
			Name:            c.Name,
			SetCode:         payload.Data.Code,
			CollectorNumber: c.Number,
			ManaCost:        c.ManaCost,
			ManaValue:       c.ManaValue,
			Rarity:          c.Rarity,
			Types:           c.Types,
			Subtypes:        c.Subtypes,
			OracleText:      c.Text,
			Power:           c.Power,
			Toughness:       c.Toughness,
			Colors:          c.Colors,
		}
		docs = append(docs, doc)
	}

	if len(docs) == 0 {
		log.Fatalf("No valid cards found to insert for set %s", setCode)
	}

	// =========================================================================
	// SECTION 8: BULK DATABASE INSERTION
	// =========================================================================
	fmt.Printf("⏳ [3/4] Inserting %d cards into MongoDB...\n", len(docs))
	res, err := cardCollection.InsertMany(ctx, docs)
	if err != nil {
		log.Fatalf("❌ Failed to insert cards into MongoDB: %v", err)
	}

	// =========================================================================
	// SECTION 9: DATABASE INDEXING
	// =========================================================================
	fmt.Println("⏳ [4/4] Verifying database indexes...")
	indexModel := mongo.IndexModel{
		Keys: bson.D{
			{Key: "set_code", Value: 1}, // 1 = Ascending
			{Key: "rarity", Value: 1},
		},
	}
	_, err = cardCollection.Indexes().CreateOne(ctx, indexModel)
	if err != nil {
		log.Printf("⚠️ Warning: Failed to create index: %v", err)
	}

	fmt.Printf("\n✨ SUCCESS! Imported %d cards from %s (%s) into MongoDB!\n\n", len(res.InsertedIDs), payload.Data.Name, payload.Data.Code)
}
