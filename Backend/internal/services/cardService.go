package services

import (
	"backend/internal/models"
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"math/rand/v2"
	"sync"
	"time"

	"github.com/redis/go-redis/v9"
)

// CardStore defines the data access contract required by PackService.
type CardStore interface {
	GetCardBySet(ctx context.Context, setCode string) ([]*models.Card, error)
}

// setRarityPool caches pre-partitioned rarity buckets for O(1) pack collation.
type setRarityPool struct {
	all       []*models.Card
	commons   []*models.Card
	uncommons []*models.Card
	rares     []*models.Card
	mythics   []*models.Card
}

// PackService coordinates booster pack collation, in-memory set caching, and event streaming.
type PackService struct {
	cardRepo CardStore
	rdb      *redis.Client
	cacheMu  sync.RWMutex
	setCache map[string]*setRarityPool
}

// NewPackService constructs a PackService instance with an empty in-memory set cache.
func NewPackService(cardRepo CardStore, rdb *redis.Client) *PackService {
	return &PackService{
		cardRepo: cardRepo,
		rdb:      rdb,
		setCache: make(map[string]*setRarityPool),
	}
}

// OpenPack collates a 14-card booster pack using MTG rarity distribution rules:
// - 1 Rare or Mythic (1:8 chance of upgrading to Mythic)
// - 3 Uncommons (deduplicated)
// - 10 Commons (deduplicated)
// It also publishes a background event to Redis Streams for asynchronous analytics.
func (p *PackService) OpenPack(ctx context.Context, setCode string, userID string) ([]*models.Card, error) {
	pool, err := p.getSetPool(ctx, setCode)
	if err != nil {
		return nil, err
	}
	if len(pool.all) == 0 {
		return nil, errors.New("no cards found for set: " + setCode)
	}

	// 1. Collate pack directly from pre-partitioned rarity buckets
	pack := make([]*models.Card, 0, 14)
	if len(pool.mythics) > 0 && rand.IntN(8) == 0 {
		pack = append(pack, pickUnique(pool.mythics, 1)...)
	} else if len(pool.rares) > 0 {
		pack = append(pack, pickUnique(pool.rares, 1)...)
	}
	pack = append(pack, pickUnique(pool.uncommons, 3)...)
	pack = append(pack, pickUnique(pool.commons, 10)...)

	// 2. Emit asynchronous pack opening event to Redis Streams
	if userID == "" {
		userID = "anonymous"
	}
	p.publishPackOpenedEvent(userID, setCode, pack)

	return pack, nil
}

func pickUnique(pool []*models.Card, count int) []*models.Card {
	if len(pool) <= count {
		out := make([]*models.Card, len(pool))
		copy(out, pool)
		return out
	}
	perm := rand.Perm(len(pool))
	result := make([]*models.Card, count)
	for i := range count {
		result[i] = pool[perm[i]]
	}
	return result
}

// getSetPool uses Double-Checked Locking to prevent cache stampedes (Thundering Herd)
// when multiple goroutines open a cold set concurrently, and pre-partitions cards by rarity once.
func (p *PackService) getSetPool(ctx context.Context, setCode string) (*setRarityPool, error) {
	p.cacheMu.RLock()
	cached, found := p.setCache[setCode]
	p.cacheMu.RUnlock()
	if found {
		return cached, nil
	}

	p.cacheMu.Lock()
	defer p.cacheMu.Unlock()

	// Double-check after acquiring write lock in case another goroutine populated the cache while we waited
	if cached, found = p.setCache[setCode]; found {
		return cached, nil
	}

	cards, err := p.cardRepo.GetCardBySet(ctx, setCode)
	if err != nil {
		return nil, err
	}

	pool := &setRarityPool{
		all: cards,
	}
	for _, card := range cards {
		switch card.Rarity {
		case "common":
			pool.commons = append(pool.commons, card)
		case "uncommon":
			pool.uncommons = append(pool.uncommons, card)
		case "rare":
			pool.rares = append(pool.rares, card)
		case "mythic":
			pool.mythics = append(pool.mythics, card)
		}
	}

	p.setCache[setCode] = pool
	return pool, nil
}

// publishPackOpenedEvent pushes a pack opening record to Redis Streams asynchronously.
func (p *PackService) publishPackOpenedEvent(userID, setCode string, cards []*models.Card) {
	if p.rdb == nil {
		return
	}

	cardIDs := make([]string, len(cards))
	rarities := make([]string, len(cards))
	for i, c := range cards {
		cardIDs[i] = c.ScryfallID
		rarities[i] = c.Rarity
	}

	cardIDsJSON, err := json.Marshal(cardIDs)
	if err != nil {
		slog.Warn("Failed to serialize card IDs for stream", "err", err)
		return
	}

	raritiesJSON, err := json.Marshal(rarities)
	if err != nil {
		slog.Warn("Failed to serialize rarities for stream", "err", err)
		return
	}

	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()

		err := p.rdb.XAdd(ctx, &redis.XAddArgs{
			Stream: "stream:pack_openings",
			ID:     "*",
			Values: map[string]any{
				"user_id":    userID,
				"set_code":   setCode,
				"card_ids":   string(cardIDsJSON),
				"rarities":   string(raritiesJSON),
				"card_count": len(cards),
				"opened_at":  time.Now().UTC().Format(time.RFC3339),
			},
		}).Err()
		if err != nil {
			slog.Warn("Failed to publish pack opening to Redis stream", "set_code", setCode, "err", err)
		}
	}()
}
