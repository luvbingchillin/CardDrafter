package services

import (
	"backend/internal/models"
	"context"
	"math/rand/v2"
	"runtime"
	"sync"
)

type workerSummary struct {
    mythics   int
    rares     int
    uncommons int
    commons   int
}

func RunSimulation(ctx context.Context, setCode string, p *PackService, totalPacks int)  (*models.SimulationResult, error){
	cards, err := p.cardRepo.GetCardBySet(ctx, setCode)
	if err!=nil{
		return nil, err
	}
	numWorkers := runtime.NumCPU()
	packsPerWorkers := totalPacks/numWorkers
	remainder:=totalPacks%numWorkers
	resultsChan := make(chan workerSummary, numWorkers)
	var wg sync.WaitGroup
	for i:= range numWorkers{
		packsToRun := packsPerWorkers
		if i==0{
			packsToRun += remainder
		}
		wg.Add(1)
		go func(count int){
			defer wg.Done()
			var localSummary workerSummary
			for range count{
				if rand.IntN(8) == 0 {
					localSummary.mythics++
				} else {
					localSummary.rares++
				}
				localSummary.uncommons += 3
				localSummary.commons += 10
			}
			resultsChan<-localSummary
		} (packsToRun)
	}
	go func() {
		wg.Wait()
		close(resultsChan)
	}()
	var finalResult models.SimulationResult
	for summary := range resultsChan {
		finalResult.MythicCount += summary.mythics
		finalResult.RareCount += summary.rares
		finalResult.UncommonCount += summary.uncommons
		finalResult.CommonCount += summary.commons
	}

 return &finalResult, nil
}