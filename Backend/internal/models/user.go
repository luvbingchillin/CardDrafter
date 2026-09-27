package models

import (
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
)

type User struct {
	ID           bson.ObjectID `bson:"_id,omitempty" json:"id"`
	Username     string        `bson:"username"      json:"username"`
	Email        string        `bson:"email"         json:"email"`
	PasswordHash string        `bson:"password_hash" json:"-"`
	CreatedAt    time.Time     `bson:"created_at"    json:"created_at"`
}

type LoginRequest struct {
	Login    string `json:"login"`
	Password string `json:"password"`
}

type UserSummary struct {
	ID       string
	Username string
	Email    string
}

type AuthResponse struct {
	User    UserSummary
	Message string
}

type RegisterRequest struct {
	Username string `json:"username"`
	Email    string `json:"email"`
	Password string `json:"password"`
}
