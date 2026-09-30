package repository

import (
	"backend/internal/models"
	"context"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
)

type CardRepository struct {
	collection *mongo.Collection
}

func NewCardRepository(db *mongo.Database) *CardRepository {
	repo := &CardRepository{
		collection: db.Collection("cards"),
	}
	return repo
}

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
