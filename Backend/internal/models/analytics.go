package models

import (
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
)

type SetAnalytics struct {
	ID               bson.ObjectID `bson:"_id,omitempty"        json:"id"`
	SetCode          string        `bson:"set_code"            json:"set_code"`
	TotalPacksOpened int64         `bson:"total_packs_opened"  json:"total_packs_opened"`
	MythicCount      int64         `bson:"mythic_count"        json:"mythic_count"`
	RareCount        int64         `bson:"rare_count"          json:"rare_count"`
	UncommonCount    int64         `bson:"uncommon_count"      json:"uncommon_count"`
	CommonCount      int64         `bson:"common_count"        json:"common_count"`
	LastUpdated      time.Time     `bson:"last_updated"        json:"last_updated"`
}
