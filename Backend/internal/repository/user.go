package repository

import (
	"backend/internal/models"
	"context"
	"errors"
	"log"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

type UserRepository struct {
	collection *mongo.Collection
}

func NewUserRepository(db *mongo.Database) *UserRepository {
	repo := &UserRepository{
		collection: db.Collection("users"),
	}
	repo.ensureIndexes(context.Background())
	return repo
}

func (r *UserRepository) ensureIndexes(ctx context.Context) {
	indexes := []mongo.IndexModel{
		{
			Keys:    bson.D{{Key: "email", Value: 1}},
			Options: options.Index().SetUnique(true),
		},
		{
			Keys:    bson.D{{Key: "username", Value: 1}},
			Options: options.Index().SetUnique(true),
		},
	}

	// Tell Mongo collection to create these indexes if they don't already exist
	_, err := r.collection.Indexes().CreateMany(ctx, indexes)
	if err != nil {
		log.Printf("Warning: Failed to create user indexes: %v", err)
	}
}

func (r *UserRepository) CreateUser(ctx context.Context, user *models.User) error {
	result, err := r.collection.InsertOne(ctx, user)
	if err != nil {
		return err
	}
	if oid, ok := result.InsertedID.(bson.ObjectID); ok {
		user.ID = oid
	}
	return nil
}

func (r *UserRepository) FindByLogin(ctx context.Context, login string) (*models.User, error) {
	filter := bson.M{
		"$or": []bson.M{
			{"email": login},
			{"username": login},
		},
	}

	var user models.User
	err := r.collection.FindOne(ctx, filter).Decode(&user)
	if err != nil {
		// If no user was found, return nil without treating it as a system crash
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil, nil
		}
		return nil, err
	}

	return &user, nil
}

func (r *UserRepository) FindByGoogleId(ctx context.Context, googID string) (*models.User, error) {
	var user models.User
	err := r.collection.FindOne(ctx, bson.M{"google_id": googID}).Decode(&user)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil, nil
		}
		return nil, err
	}
	return &user, nil
}
