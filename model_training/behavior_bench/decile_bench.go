package main

import (
	"encoding/json"
	"fmt"
	"math"
	"os"
	"sort"
	"strconv"

	"github.com/dgray001/gray_online/game"
	"github.com/dgray001/gray_online/game/games/fiddlesticks"
)

type DialConfig struct {
	Param       string  `json:"param"`
	MinVal      float64 `json:"min_val"`
	MaxVal      float64 `json:"max_val"`
	IsBetting   bool    `json:"is_betting"`
	Description string  `json:"description"`
}

type DecileDistribution struct {
	Param             string    `json:"param"`
	Model             string    `json:"model"`
	TotalCurated      int       `json:"total_curated"`
	TotalFlipped      int       `json:"total_flipped"`
	FlipRatePct       float64   `json:"flip_rate_pct"`
	Deciles           [10]int   `json:"deciles"` // [0-10%, 10-20%, ..., 90-100%]
	DecilePercentages [10]string `json:"decile_percentages"`
	SpreadScore       float64   `json:"spread_score"` // Entropy / uniformity measure (1.0 = perfectly flat)
	KeyExamples       []Example `json:"key_examples"`
}

type Example struct {
	Decile      string  `json:"decile"`
	ThresholdUI float64 `json:"threshold_ui"`
	RawVal      float64 `json:"raw_val"`
	BeforePlay  string  `json:"before_play"`
	AfterPlay   string  `json:"after_play"`
	Context     string  `json:"context"`
}

var dials = []DialConfig{
	{Param: "overtrick_aversion", MinVal: 0.0, MaxVal: 2.5, IsBetting: false, Description: "Duck tricks when at contract"},
	{Param: "undertrick_aversion", MinVal: 0.0, MaxVal: 3.5, IsBetting: false, Description: "Win tricks aggressively when behind contract"},
	{Param: "trump_hoarding", MinVal: 0.0, MaxVal: 1.0, IsBetting: false, Description: "Preserve trumps vs bleed/ruff early"},
	{Param: "void_hunting", MinVal: 0.0, MaxVal: 1.0, IsBetting: false, Description: "Discard off-suits to open ruffing channels"},
	{Param: "vindictiveness", MinVal: 0.0, MaxVal: 2.0, IsBetting: false, Description: "Sabotage leader contracts when own contract secure"},
	{Param: "bid_trust", MinVal: 0.0, MaxVal: 2.5, IsBetting: false, Description: "Trust opponent bids correlate with card distribution"},
	{Param: "bet_optimism", MinVal: -0.60, MaxVal: 0.40, IsBetting: true, Description: "Rounding offset during bidding"},
}

func main() {
	fmt.Println("=== Running Decile Distribution Analysis on Models 4a & 4b ===")

	models := []struct {
		ID   uint8
		Name string
	}{
		{ID: 5, Name: "Model 4a"},
		{ID: 6, Name: "Model 4b"},
	}

	results := make(map[string]map[string]*DecileDistribution)

	for _, m := range models {
		fmt.Printf("\n=======================================================\n")
		fmt.Printf(" MODEL: %s\n", m.Name)
		fmt.Printf("=======================================================\n")
		results[m.Name] = make(map[string]*DecileDistribution)

		for _, d := range dials {
			if d.Param == "bid_trust" && m.ID == 5 {
				continue
			}

			dist := analyzeDialDeciles(m.ID, m.Name, d)
			results[m.Name][d.Param] = dist

			fmt.Printf("\n--- Dial: %s (Total Flipped: %d/%d = %.1f%%) ---\n",
				d.Param, dist.TotalFlipped, dist.TotalCurated, dist.FlipRatePct)
			fmt.Printf("Decile Counts [0-10%% -> 90-100%%]: %v\n", dist.Deciles)
			fmt.Println("Decile Spread:")
			for i := 0; i < 10; i++ {
				bar := renderBar(dist.Deciles[i], 15)
				fmt.Printf("  %2d0%%-%2d0%% : %3d (%5.1f%%) %s\n",
					i, i+1, dist.Deciles[i],
					float64(dist.Deciles[i])/float64(dist.TotalFlipped)*100, bar)
			}
			fmt.Printf("Spread Uniformity Score (0 to 1): %.2f\n", dist.SpreadScore)
		}
	}

	_ = os.MkdirAll("outputs/behavior_bench", 0755)
	b, _ := json.MarshalIndent(results, "", "  ")
	_ = os.WriteFile("outputs/behavior_bench/decile_distribution_report.json", b, 0644)
	fmt.Println("\nSaved full decile report to outputs/behavior_bench/decile_distribution_report.json")
}

func analyzeDialDeciles(modelID uint8, modelName string, dial DialConfig) *DecileDistribution {
	situations := generateCuratedSituations(dial)
	total := len(situations)
	flippedCount := 0
	var deciles [10]int
	var switchPoints []float64

	for _, sit := range situations {
		// Evaluate action at UI = 0.0 and UI = 1.0
		lowAction := evalAction(modelID, dial.Param, dial.MinVal, sit, dial.IsBetting)
		highAction := evalAction(modelID, dial.Param, dial.MaxVal, sit, dial.IsBetting)

		if lowAction != highAction {
			flippedCount++

			// Binary search to find exact switch threshold in UI space [0.0, 1.0]
			lowUI := 0.0
			highUI := 1.0
			for step := 0; step < 16; step++ {
				midUI := (lowUI + highUI) / 2.0
				rawVal := dial.MinVal + midUI*(dial.MaxVal-dial.MinVal)
				actMid := evalAction(modelID, dial.Param, rawVal, sit, dial.IsBetting)
				if actMid == lowAction {
					lowUI = midUI
				} else {
					highUI = midUI
				}
			}
			exactUI := (lowUI + highUI) / 2.0
			switchPoints = append(switchPoints, exactUI)

			// Bin into decile [0..9]
			decileIdx := int(exactUI * 10)
			if decileIdx >= 10 {
				decileIdx = 9
			} else if decileIdx < 0 {
				decileIdx = 0
			}
			deciles[decileIdx]++
		}
	}

	// Calculate uniformity score (normalized Shannon entropy)
	spreadScore := 0.0
	if flippedCount > 0 {
		entropy := 0.0
		for _, count := range deciles {
			if count > 0 {
				p := float64(count) / float64(flippedCount)
				entropy -= p * math.Log2(p)
			}
		}
		maxEntropy := math.Log2(10.0) // ~3.32
		spreadScore = entropy / maxEntropy
	}

	var decilePcts [10]string
	for i := 0; i < 10; i++ {
		pct := 0.0
		if flippedCount > 0 {
			pct = float64(deciles[i]) / float64(flippedCount) * 100.0
		}
		decilePcts[i] = fmt.Sprintf("%.1f%%", pct)
	}

	sort.Float64s(switchPoints)

	rate := 0.0
	if total > 0 {
		rate = float64(flippedCount) / float64(total) * 100.0
	}

	return &DecileDistribution{
		Param:             dial.Param,
		Model:             modelName,
		TotalCurated:      total,
		TotalFlipped:      flippedCount,
		FlipRatePct:       rate,
		Deciles:           deciles,
		DecilePercentages: decilePcts,
		SpreadScore:       spreadScore,
	}
}

type CuratedState struct {
	NumPlayers int
	Round      uint8
	TrumpSuit  int
	SeedOffset int
}

func generateCuratedSituations(dial DialConfig) []CuratedState {
	// Generate 120 curated states spanning tight, non-trivial decision nodes
	var list []CuratedState
	players := []int{2, 3, 4, 5, 6, 7, 8}
	rounds := []uint8{2, 3, 4, 5, 6, 7, 8, 9, 10}

	for i := 0; len(list) < 110; i++ {
		pc := players[i%len(players)]
		r := rounds[(i/len(players))%len(rounds)]
		if int(r)*pc > 51 {
			continue
		}
		list = append(list, CuratedState{
			NumPlayers: pc,
			Round:      r,
			TrumpSuit:  i % 4,
			SeedOffset: i * 7,
		})
	}
	return list
}

func evalAction(modelID uint8, param string, rawVal float64, sit CuratedState, isBetting bool) int {
	gameBase := &game.GameBase{
		GameEndedChannel: make(chan string, 1),
	}
	f := fiddlesticks.InitializeGame(gameBase)

	params := map[string]string{
		param: strconv.FormatFloat(rawVal, 'f', 4, 64),
	}
	f.AddInternalAiPlayer(modelID, params)

	for i := 1; i < sit.NumPlayers; i++ {
		f.AddInternalAiPlayer(3, nil) // Model 3 opponents
	}

	f.SetSettings(sit.Round, sit.Round, 10, 3, 1)
	f.GetBase().StartGame()
	f.StartAiGame(true)

	player0 := f.GetFiddlesticksPlayer(0)
	if isBetting {
		return int(fiddlesticks.GetAiBid(player0, f))
	}

	for f.IsBetting() {
		f.ExecuteAiTurn(false)
	}

	return fiddlesticks.GetAiPlayCard(player0, f)
}

func renderBar(count int, maxLen int) string {
	chars := ""
	barLen := count
	if barLen > maxLen {
		barLen = maxLen
	}
	for i := 0; i < barLen; i++ {
		chars += "█"
	}
	return chars
}
