package repository

import (
	"backend/internal/models"
	"context"
	"errors"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
)

// AnalyticsRepository provides access to aggregated pack opening statistics.
type AnalyticsRepository struct {
	collection *mongo.Collection
}

// NewAnalyticsRepository initializes an AnalyticsRepository targeting 'set_analytics'.
func NewAnalyticsRepository(db *mongo.Database) *AnalyticsRepository {
	return &AnalyticsRepository{
		collection: db.Collection("set_analytics"),
	}
}

// GetAnalyticsBySet retrieves pull analytics for a specific set code.
// If no packs have been opened for the set yet, it returns a zeroed struct instead of an error.
func (r *AnalyticsRepository) GetAnalyticsBySet(ctx context.Context, setCode string) (*models.SetAnalytics, error) {
	var analytics models.SetAnalytics
	err := r.collection.FindOne(ctx, bson.M{"set_code": setCode}).Decode(&analytics)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			// Graceful fallback for sets with 0 packs opened
			return &models.SetAnalytics{
				SetCode: setCode,
			}, nil
		}
		return nil, err
	}
	return &analytics, nil
}
