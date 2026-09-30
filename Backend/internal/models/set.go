package models

import (
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
)

type Set struct {
	ID          bson.ObjectID `bson:"_id,omitempty" json:"id"`
	Code        string        `bson:"code"           json:"code"`         // "BLB"
	Name        string        `bson:"name"           json:"name"`         // "Bloomburrow"
	ReleaseDate string        `bson:"release_date"   json:"release_date"` // "2024-08-02"
	TotalCards  int           `bson:"total_cards"    json:"total_cards"`  // 261
	PackImage   string        `bson:"pack_image"     json:"pack_image"`   // "/packs/BLB.jpg"
	CreatedAt   time.Time     `bson:"created_at"     json:"created_at"`
}
