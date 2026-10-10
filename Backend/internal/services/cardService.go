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


// PackService coordinates booster pack collation, in-memory set caching, and event streaming.
type PackService struct {
	cardRepo *repository.CardRepository
	rdb      *redis.Client
	cacheMu  sync.RWMutex
	setCache map[string][]*models.Card
}

// NewPackService constructs a PackService instance with an empty in-memory set cache.
func NewPackService(cardRepo *repository.CardRepository, rdb *redis.Client) *PackService {
	return &PackService{
		cardRepo: cardRepo,
		rdb:      rdb,
		setCache: make(map[string][]*models.Card),
	}
}

// OpenPack collates a 14-card booster pack using MTG rarity distribution rules:
// - 1 Rare or Mythic (1:8 chance of upgrading to Mythic)
// - 3 Uncommons (deduplicated)
// - 10 Commons (deduplicated)
// It also publishes a background event to Redis Streams for asynchronous analytics.
func (p *PackService) OpenPack(ctx context.Context, setCode string, userID string) ([]*models.Card, error) {
	allCards, err := p.getSetCards(ctx, setCode)
	if err != nil {
		return nil, err
	}
	if len(allCards) == 0 {
		return nil, errors.New("no cards found for set: " + setCode)
	}

	// 1. Group set card pool by rarity
	var c, uc, r, m []*models.Card
	for _, card := range allCards {
		switch card.Rarity {
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

	// 2. Collate pack according to MTG booster slot ratios
	var pack []*models.Card
	if len(m) > 0 && rand.IntN(8) == 0 {
		pack = append(pack, pickUnique(m, 1)...)
	} else if len(r) > 0 {
		pack = append(pack, pickUnique(r, 1)...)
	}
	pack = append(pack, pickUnique(uc, 3)...)
	pack = append(pack, pickUnique(c, 10)...)

	// 3. Emit asynchronous pack opening event to Redis Streams
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