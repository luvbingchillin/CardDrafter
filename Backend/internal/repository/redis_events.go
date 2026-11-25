package repository

import (
	"backend/internal/draft"
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

const draftStreamTTL = 24 * time.Hour

// Compile-time interface satisfaction check.
var _ draft.EventStore = (*RedisEventStore)(nil)

type RedisEventStore struct {
	rdb *redis.Client
}

func NewRedisEventStore(rdb *redis.Client) *RedisEventStore {
	if rdb == nil {
		return nil
	}
	return &RedisEventStore{rdb: rdb}
}

func (s *RedisEventStore) streamKey(roomID string) string {
	return fmt.Sprintf("stream:draft:%s", roomID)
}

func (s *RedisEventStore) Append(ctx context.Context, roomID string, event draft.DraftEvent) error {
	if s == nil || s.rdb == nil {
		return nil
	}
	jsonBytes, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("failed to marshal draft event: %w", err)
	}

	key := s.streamKey(roomID)
	args := redis.XAddArgs{
		Stream: key,
		ID:     "*",
		Values: map[string]any{
			"type":    string(event.Type),
			"version": event.Version,
			"data":    string(jsonBytes),
		},
	}
	if err := s.rdb.XAdd(ctx, &args).Err(); err != nil {
		return fmt.Errorf("failed to append event to stream: %w", err)
	}

	// Sliding 24-hour TTL so completed or abandoned room streams self-clean from Redis RAM.
	if err := s.rdb.Expire(ctx, key, draftStreamTTL).Err(); err != nil {
		return fmt.Errorf("failed to set TTL on stream %s: %w", key, err)
	}

	return nil
}

func (s *RedisEventStore) LoadEvents(ctx context.Context, roomID string) ([]draft.DraftEvent, error) {
	if s == nil || s.rdb == nil {
		return nil, nil
	}
	messages, err := s.rdb.XRange(ctx, s.streamKey(roomID), "-", "+").Result()
	if err != nil {
		return nil, fmt.Errorf("failed to load events from stream: %w", err)
	}
	events := make([]draft.DraftEvent, 0, len(messages))
	for _, msg := range messages {
		raw, ok := msg.Values["data"]
		if !ok {
			return nil, fmt.Errorf("message %s missing 'data' field", msg.ID)
		}
		dataStr, ok := raw.(string)
		if !ok {
			return nil, fmt.Errorf("message %s 'data' is not a string", msg.ID)
		}
		var event draft.DraftEvent
		if err := json.Unmarshal([]byte(dataStr), &event); err != nil {
			return nil, fmt.Errorf("failed to unmarshal event %s: %w", msg.ID, err)
		}
		events = append(events, event)
	}
	return events, nil
}
