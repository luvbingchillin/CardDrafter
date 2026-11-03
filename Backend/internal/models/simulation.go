package models

// SimulationRequest defines the parameters for a Monte Carlo booster opening run.
type SimulationRequest struct {
	SetCode   string `json:"setCode"`
	PackCount int    `json:"packCount"` // e.g. 1000 to 100000
}

// SimulationResult captures empirical counts, observed percentages, and elapsed time.
type SimulationResult struct {
	SetCode       string             `json:"setCode"`
	PacksSimulated int               `json:"packsSimulated"`
	ElapsedMs     int64              `json:"elapsedMs"`
	MythicCount   int                `json:"mythicCount"`
	RareCount     int                `json:"rareCount"`
	UncommonCount int                `json:"uncommonCount"`
	CommonCount   int                `json:"commonCount"`
	MythicRate    float64            `json:"mythicRate"`    // e.g. 0.1248 (expected ~0.125)
	RareRate      float64            `json:"rareRate"`      // e.g. 0.8752
	RarestPulls   map[string]int     `json:"rarestPulls"`   // top mythic card counts
}