package draft

import (
	"log"
	"sync"
)

type Hub struct {
	mu          sync.RWMutex
	rooms       map[string]*Room
	packService PackOpener
	eventStore  EventStore
}

func NewHub(packService PackOpener, eventStore EventStore) *Hub {
	return &Hub{
		packService: packService,
		eventStore:  eventStore,
		rooms:       make(map[string]*Room),
	}
}

func (h *Hub) GetOrCreateRoom(id string) *Room {
	h.mu.Lock()
	defer h.mu.Unlock()
	if room, ok := h.rooms[id]; ok {
		log.Printf("Room with id %s already active in Hub", id)
		return room
	}

	room := NewRoom(id, h.packService, h.eventStore, h)
	h.rooms[id] = room
	go room.Run()
	return room
}

func (h *Hub) RemoveRoom(id string) {
	h.mu.Lock()
	defer h.mu.Unlock()
	delete(h.rooms, id)
}
