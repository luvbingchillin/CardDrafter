package handlers

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"time"

	"github.com/redis/go-redis/v9"
	"go.mongodb.org/mongo-driver/v2/mongo"
)

type HealthHandler struct {
	mongoClient *mongo.Client
	rdb *redis.Client
}

func NewHealthHandler(mongoClient *mongo.Client, rdb *redis.Client) *HealthHandler{
	return &HealthHandler{mongoClient: mongoClient, rdb: rdb}
}

func (h *HealthHandler) Liveness(w http.ResponseWriter, r *http.Request){
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(map[string]string{"status":"ok"})
}

func (h *HealthHandler) Readiness(w http.ResponseWriter, r *http.Request){
	ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
	defer cancel()
	checks := map[string]string{"mongodb": "ok", "redis": "ok"} 
	ready := true
	if h.mongoClient == nil{
		checks["mongodb"] = "unavailable"
		ready = false
	} else if err := h.mongoClient.Ping(ctx, nil); err != nil{
		slog.Error("Unable to ping mongoDB client")
		checks["mongodb"] = "error"
		ready = false
	}
	if h.rdb == nil{
		checks["redis"] = "unavailable"
		ready = false
	} else if err := h.rdb.Ping(ctx).Err(); err != nil{
		slog.Error("Unable to ping Redis client")
		checks["redis"] = "error"
		ready = false
	}
	w.Header().Set("Content-Type","application/json")
	var status int
	var statusText string
	if ready == true{
		status = http.StatusOK
		statusText = "ready"
	} else {
		status = http.StatusServiceUnavailable
		statusText = "not_ready"
	}
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(map[string]any{"status":statusText, "checks":checks})
}