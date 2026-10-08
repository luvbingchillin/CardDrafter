package services

import (
	"backend/internal/models"
	"backend/internal/repository"
	"context"
	"encoding/json"
	"errors"
	"log"
	"math/rand/v2"
	"sync"
	"time"

	"github.com/redis/go-redis/v9"
)


type PackService struct{
	cardRepo *repository.CardRepository
	rdb *redis.Client
	cacheMu sync.RWMutex
	setCache map[string][]*models.Card
}


func NewPackService(cardRepo *repository.CardRepository, rdb *redis.Client) *PackService{
	return &PackService{cardRepo: cardRepo, rdb: rdb, setCache: make(map[string][]*models.Card)}
}

func (p *PackService) OpenPack(ctx context.Context, setCode string, userID string) ([]*models.Card, error){
	allCards, err := p.getSetCards(ctx, setCode)
	if err != nil{
		return nil, err
	}
	if len(allCards)==0{
		return nil, errors.New("No cards found for" + setCode)
	}
	var c, uc, r, m []*models.Card
	for _,card := range allCards{
		switch card.Rarity{
		case "common":
			c = append(c, card)
		case "uncommon":
			uc = append(uc, card)
		case "rare":
			r = append(r, card)
		case "mythic":
			m = append(m, card)
		}
	}
	var pack []*models.Card
	if len(m) > 0 && rand.IntN(8) == 0 {
		pack = append(pack, pickUnique(m, 1)...)
	} else if len(r) > 0 {
		pack = append(pack, pickUnique(r, 1)...)
	}
	// 4. Pick 3 Uncommons without duplicates
	pack = append(pack, pickUnique(uc, 3)...)
	// 5. Pick 10 Commons without duplicates
	pack = append(pack, pickUnique(c, 10)...)

	// Publish analytics event to Redis Stream
	if userID == "" {
		userID = "anonymous"
	}
	p.publishPackOpenedEvent(userID, setCode, pack)

	return pack, nil
}

func pickUnique(pool []*models.Card, count int) []*models.Card{
	if len(pool)<=(count){
		return pool
	}
	perm := rand.Perm(len(pool))
	result := make([]*models.Card, count)
	for i:=0;i<count;i++{
		result[i] = pool[perm[i]]
	}
	return result
}


func (p *PackService) getSetCards(ctx context.Context, setCode string)([]*models.Card, error){
	p.cacheMu.RLock()
	cached, found := p.setCache[setCode]
	p.cacheMu.RUnlock()
	if found {
		return cached, nil
	}
	cards, err := p.cardRepo.GetCardBySet(ctx, setCode)
	if err != nil{
		return nil, err
	}
	p.cacheMu.Lock()
	p.setCache[setCode] = cards
	p.cacheMu.Unlock()
	return cards, nil
}

// publishPackOpenedEvent pushes a pack opening record to Redis Streams asynchronously.
func (p *PackService) publishPackOpenedEvent(userID, setCode string, cards []*models.Card) {
	if p.rdb == nil {
		return
	}

	cardIDs := make([]string, len(cards))
	for i, c := range cards {
		cardIDs[i] = c.ScryfallID
	}
	cardIDsJSON, err := json.Marshal(cardIDs)
	if err != nil {
		log.Printf("Warning: failed to serialize card IDs for stream: %v", err)
		return
	}

	// Detached background goroutine with its own timeout so the HTTP client is never blocked
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()

		err := p.rdb.XAdd(ctx, &redis.XAddArgs{
			Stream: "stream:pack_openings",
			ID:     "*",
			Values: map[string]interface{}{
				"user_id":    userID,
				"set_code":   setCode,
				"card_ids":   string(cardIDsJSON),
				"card_count": len(cards),
				"opened_at":  time.Now().UTC().Format(time.RFC3339),
			},
		}).Err()

		if err != nil {
			log.Printf("Warning: failed to publish pack opening to Redis stream: %v", err)
		}
	}()
}