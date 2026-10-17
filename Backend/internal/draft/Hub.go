package draft

import (
	"log"
	"sync"
)

type Hub struct {
	mu sync.RWMutex
	rooms map[string]*Room
}

func NewHub() *Hub{
	return &Hub{rooms:make(map[string]*Room)}
}

func (h *Hub) GetOrCreateRoom(id string,) *Room{
	h.mu.Lock()
	defer h.mu.Unlock()
	if _,ok:=h.rooms[id]; ok{
		log.Printf("Room with id %s already exists", id)
		return h.rooms[id]
	} else {
		room := NewRoom(id)
		h.rooms[id] = room
		go room.Run()
		return room
	}
}
	
func (h *Hub) RemoveRoom(id string){
	h.mu.Lock()
	defer h.mu.Unlock()
	if _,ok:=h.rooms[id];ok{
		delete(h.rooms, id)
	}
}