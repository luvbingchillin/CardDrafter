package draft

import (
	"backend/internal/models"
	"encoding/json"
	"time"
)

type EventType string

const (
	EventRoomCreated    EventType = "ROOM_CREATED"
	EventPlayerJoined   EventType = "PLAYER_JOINED"
	EventPlayerLeft     EventType = "PLAYER_LEFT"
	EventReadyToggled   EventType = "READY_TOGGLED"
	EventPacksSet       EventType = "PACKS_SET"
	EventRoundStarted   EventType = "ROUND_STARTED"
	EventCardPicked     EventType = "CARD_PICKED"
	EventPackPassed     EventType = "PACK_PASSED"
	EventDraftCompleted EventType = "DRAFT_COMPLETED"
)

type DraftEvent struct {
	ID        string          `json:"id"`
	RoomID    string          `json:"roomId"`
	Type      EventType       `json:"type"`
	Version   int             `json:"version"`
	Timestamp time.Time       `json:"timestamp"`
	Payload   json.RawMessage `json:"payload"`
}

type RoomCreatedPayload struct {
	HostID string `json:"hostId"`
}

type PlayerJoinedPayload struct {
	SeatIndex int    `json:"seatIndex"`
	UserID    string `json:"userId"`
}

type PlayerLeftPayload struct {
	UserID string `json:"userId"`
}

type ReadyToggledPayload struct {
	UserID  string `json:"userId"`
	IsReady bool   `json:"isReady"`
}

type PacksSetPayload struct {
	PackSets []string `json:"packSets"`
}

type RoundStartedPayload struct {
	Round     int                    `json:"round"`
	SetCode   string                 `json:"setCode"`
	SeatPacks map[int][]*models.Card `json:"seatPacks"`
}

type CardPickedPayload struct {
	SeatIndex  int    `json:"seatIndex"`
	UserID     string `json:"userId"`
	CardID     string `json:"cardId"`
	IsAutoPick bool   `json:"isAutoPick"`
}

type PackPassedPayload struct {
	FromSeatIndex  int            `json:"fromSeatIndex"`
	ToSeatIndex    int            `json:"toSeatIndex"`
	RemainingCards []*models.Card `json:"remainingCards"`
}

type DraftCompletedPayload struct {
	CompletedAt time.Time `json:"completedAt"`
}
