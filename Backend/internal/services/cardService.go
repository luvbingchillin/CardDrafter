package services

import (
	"backend/internal/models"
	"backend/internal/repository"
	"context"
	"errors"
	"math/rand/v2"
	"sync"
)


type PackService struct{
	cardRepo *repository.CardRepository
	cacheMu sync.RWMutex
	setCache map[string][]*models.Card
}


func NewPackService(cardRepo *repository.CardRepository) *PackService{
	return &PackService{cardRepo: cardRepo, setCache:make(map[string][]*models.Card), }

}

func (p *PackService) OpenPack(ctx context.Context, setCode string) ([]*models.Card, error){
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