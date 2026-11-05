package models

// SimulationRequest defines the parameters for a Monte Carlo booster opening run.
// Note: setCode is passed via the URL path (/api/packs/{setCode}/simulate),
// so the request body contains strictly action parameters.
type SimulationRequest struct {
	PackCount int `json:"packCount"`
}

// SimulationResult captures empirical counts, observed percentages, and elapsed time.
type SimulationResult struct {
	SetCode        string         `json:"setCode"`
	PacksSimulated int            `json:"packsSimulated"`
	ElapsedMs      int64          `json:"elapsedMs"`
	MythicCount    int            `json:"mythicCount"`
	RareCount      int            `json:"rareCount"`
	UncommonCount  int            `json:"uncommonCount"`
	CommonCount    int            `json:"commonCount"`
	MythicRate     float64        `json:"mythicRate"`
	RareRate       float64        `json:"rareRate"`
	RarestPulls    map[string]int `json:"rarestPulls"`
}
