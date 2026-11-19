package repository

import (
	"backend/internal/draft"
	"context"
	"encoding/json"
	"fmt"

	"github.com/redis/go-redis/v9"
)

type RedisEventStore struct {
	rdb *redis.Client
}


func NewRedisEventStore(rdb *redis.Client) *RedisEventStore{
	return &RedisEventStore{rdb:rdb}
}

func (s *RedisEventStore) streamKey(roomID string) string{
	return fmt.Sprintf("stream:draft:%s", roomID)
}

func (s *RedisEventStore) Append(ctx context.Context, roomId string, event draft.DraftEvent) error{
	jsonBytes, err := json.Marshal(event)
	if err != nil{
		return fmt.Errorf("failed to marshal draft event: %w", err)
	}
	args := redis.XAddArgs{
		Stream: s.streamKey(roomId),
		ID:"*",
		Values:map[string]any{
			"type":string(event.Type),
			"version": event.Version,
			"data":string(jsonBytes),
		},
	}
	if err := s.rdb.XAdd(ctx, &args).Err(); err != nil {
		return fmt.Errorf("failed to append event to stream: %w", err)
	}
	return nil
}

func (s *RedisEventStore) LoadEvents(ctx context.Context, roomID string) ([]draft.DraftEvent, error){
	messages, err := s.rdb.XRange(ctx, s.streamKey(roomID), "-","+").Result()
	if err!=nil{
		return nil, fmt.Errorf("XRange method failed %w", err)
	}
	events := make([]draft.DraftEvent,0,len(messages))
	for _,msg := range messages{
		raw, ok := msg.Values["data"]
		if !ok {
			return nil, fmt.Errorf("message %s missing 'data' field", msg.ID)
		}
		dataStr, ok := raw.(string)
		if !ok {
			return nil, fmt.Errorf("message %s 'data' is not a string", msg.ID)
		}
		var event draft.DraftEvent
		err := json.Unmarshal([]byte(dataStr), &event)
		if err!=nil{
			return nil, fmt.Errorf("failed to unmarshal event %s: %w", msg.ID, err)
		}
		events = append(events, event)
	}
	return events, nil

}