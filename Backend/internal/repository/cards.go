package repository

import (
	"backend/internal/models"
	"context"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
)

// CardRepository provides data access operations for card records in MongoDB.
type CardRepository struct {
	collection *mongo.Collection
}

// NewCardRepository initializes a CardRepository backed by the 'cards' collection.
func NewCardRepository(db *mongo.Database) *CardRepository {
	return &CardRepository{
		collection: db.Collection("cards"),
	}
}

// GetCardBySet retrieves all cards belonging to a specific set code.
func (c *CardRepository) GetCardBySet(ctx context.Context, setCode string) ([]*models.Card, error) {
	cursor, err := c.collection.Find(ctx, bson.M{"set_code": setCode})
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)
	var cards []*models.Card
	if err := cursor.All(ctx, &cards); err != nil {
		return nil, err
	}
	return cards, nil
}

// GetCardBySetAndRarity queries cards filtered by both set code and rarity tier.
func (c *CardRepository) GetCardBySetAndRarity(ctx context.Context, rarity string, setCode string) ([]*models.Card, error) {
	cursor, err := c.collection.Find(ctx, bson.M{"set_code": setCode, "rarity": rarity})
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)
	var cards []*models.Card
	if err := cursor.All(ctx, &cards); err != nil {
		return nil, err
	}
	return cards, nil
}
