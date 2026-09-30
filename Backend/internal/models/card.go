package models

import (
	"go.mongodb.org/mongo-driver/v2/bson"
)

type Card struct {
	ID              bson.ObjectID `bson:"_id,omitempty" json:"id"`
	UUID            string        `bson:"uuid"           json:"uuid"`        // MTGJSON unique ID
	ScryfallID      string        `bson:"scryfall_id"    json:"scryfall_id"` // Used for images
	Name            string        `bson:"name"           json:"name"`
	SetCode         string        `bson:"set_code"       json:"set_code"` // e.g. "BLB"
	CollectorNumber string        `bson:"number"         json:"number"`
	ManaCost        string        `bson:"mana_cost"      json:"mana_cost"`  // e.g. "{1}{R}"
	ManaValue       float64       `bson:"mana_value"     json:"mana_value"` // CMC e.g. 2.0
	Rarity          string        `bson:"rarity"         json:"rarity"`     // common, uncommon, rare, mythic
	Types           []string      `bson:"types"          json:"types"`      // Creature, Instant, etc.
	Subtypes        []string      `bson:"subtypes"       json:"subtypes"`   // Elf, Dragon, etc.
	OracleText      string        `bson:"oracle_text"    json:"oracle_text"`
	Power           string        `bson:"power,omitempty" json:"power,omitempty"` // String because of "*", "X"
	Toughness       string        `bson:"toughness,omitempty" json:"toughness,omitempty"`
	Colors          []string      `bson:"colors"         json:"colors"` // W, U, B, R, G
}
