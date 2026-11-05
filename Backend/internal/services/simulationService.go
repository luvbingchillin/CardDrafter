package services

import (
	"backend/internal/models"
	"context"
	"errors"
	"math/rand/v2"
	"runtime"
	"sync"
	"time"
)

// CardProvider defines the narrow data access contract required by SimulationService.
// Decouples the simulation engine from direct database or concrete repository dependencies.
type CardProvider interface {
	GetCardBySet(ctx context.Context, setCode string) ([]*models.Card, error)
}

// SimulationService coordinates multi-core Monte Carlo booster simulations.
type SimulationService struct {
	cardProvider CardProvider
}

// NewSimulationService constructs an instance of SimulationService.
func NewSimulationService(cp CardProvider) *SimulationService {
	return &SimulationService{cardProvider: cp}
}

type workerSummary struct {
	mythics   int
	rares     int
	uncommons int
	commons   int
}

// RunSimulation partitions totalPacks across runtime.NumCPU() worker goroutines,
// executing parallel pack collation and aggregating statistical distributions.
func (s *SimulationService) RunSimulation(ctx context.Context, setCode string, totalPacks int) (*models.SimulationResult, error) {
	if totalPacks <= 0 {
		return nil, errors.New("packCount must be greater than 0")
	}

	start := time.Now()

	// 1. Fetch set card pool from provider abstraction
	cards, err := s.cardProvider.GetCardBySet(ctx, setCode)
	if err != nil {
		return nil, err
	}
	if len(cards) == 0 {
		return nil, errors.New("no cards found for set: " + setCode)
	}

	// 2. Partition work evenly across hardware threads
	numWorkers := runtime.NumCPU()
	packsPerWorker := totalPacks / numWorkers
	remainder := totalPacks % numWorkers

	resultsChan := make(chan workerSummary, numWorkers)
	var wg sync.WaitGroup

	// 3. Fan-out: spawn worker goroutines
	for i := range numWorkers {
		packsToRun := packsPerWorker
		if i == 0 {
			packsToRun += remainder
		}

		wg.Add(1)
		go func(count int) {
			defer wg.Done()
			var localSummary workerSummary

			for range count {
				if rand.IntN(8) == 0 {
					localSummary.mythics++
				} else {
					localSummary.rares++
				}
				localSummary.uncommons += 3
				localSummary.commons += 10
			}

			resultsChan <- localSummary
		}(packsToRun)
	}

	// 4. Detached closer goroutine unblocks once all workers finish
	go func() {
		wg.Wait()
		close(resultsChan)
	}()

	// 5. Fan-in: aggregate local summaries into final result
	var finalResult models.SimulationResult
	for summary := range resultsChan {
		finalResult.MythicCount += summary.mythics
		finalResult.RareCount += summary.rares
		finalResult.UncommonCount += summary.uncommons
		finalResult.CommonCount += summary.commons
	}

	finalResult.SetCode = setCode
	finalResult.PacksSimulated = totalPacks
	finalResult.ElapsedMs = time.Since(start).Milliseconds()

	if totalPacks > 0 {
		finalResult.MythicRate = float64(finalResult.MythicCount) / float64(totalPacks)
		finalResult.RareRate = float64(finalResult.RareCount) / float64(totalPacks)
	}

	return &finalResult, nil
}
