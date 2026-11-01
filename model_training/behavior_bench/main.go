package main

import (
	"encoding/json"
	"fmt"
	"math"
	"os"
	"strconv"

	"github.com/dgray001/gray_online/game"
	"github.com/dgray001/gray_online/game/games/fiddlesticks"
)

type DialSpec struct {
	Param       string  `json:"param"`
	MinVal      float64 `json:"min_val"`
	MaxVal      float64 `json:"max_val"`
	IsBetting   bool    `json:"is_betting"`
	Description string  `json:"description"`
}

type ModelEvalReport struct {
	Model             string                   `json:"model"`
	ParamResults      map[string]*ParamSummary `json:"param_results"`
	TotalSituations   int                      `json:"total_situations"`
	ResponsiveCount   int                      `json:"responsive_count"`
	ResponsivenessPct float64                  `json:"responsiveness_pct"`
}

type ParamSummary struct {
	TotalTested      int     `json:"total_tested"`
	SwitchedCount    int     `json:"switched_count"`
	SensitivityScore float64 `json:"sensitivity_score"`
	RecommendedMin   float64 `json:"recommended_min"`
	RecommendedMax   float64 `json:"recommended_max"`
	KeyAnchor1UI     float64 `json:"key_anchor_1_ui"`
	KeyAnchor2UI     float64 `json:"key_anchor_2_ui"`
	KeyAnchor3UI     float64 `json:"key_anchor_3_ui"`
}

var dialSpecs = []DialSpec{
	{Param: "overtrick_aversion", MinVal: 0.0, MaxVal: 2.5, IsBetting: false, Description: "Avoid winning tricks when on contract"},
	{Param: "undertrick_aversion", MinVal: 0.0, MaxVal: 3.5, IsBetting: false, Description: "Urgency to win tricks when behind contract"},
	{Param: "trump_hoarding", MinVal: 0.0, MaxVal: 1.0, IsBetting: false, Description: "Hold trumps vs bleed/ruff early"},
	{Param: "void_hunting", MinVal: 0.0, MaxVal: 1.0, IsBetting: false, Description: "Discard off-suits to create void channels"},
	{Param: "vindictiveness", MinVal: 0.0, MaxVal: 2.0, IsBetting: false, Description: "Sabotage opponents who are making contract"},
	{Param: "bid_trust", MinVal: 0.0, MaxVal: 2.5, IsBetting: false, Description: "Trust opponent bids correlate with card strength"},
	{Param: "bet_optimism", MinVal: -0.60, MaxVal: 0.40, IsBetting: true, Description: "Bidding offset/risk tolerance"},
}

func evaluateDialSensitivity(modelID uint8, dial DialSpec) *ParamSummary {
	targetDeals := 105
	switched := 0
	observedSwitches := make([]float64, 0)

	playerCounts := []int{2, 3, 4, 5, 6, 7, 8}
	rounds := []uint8{1, 2, 3, 4, 5, 6, 7, 8, 9, 10}

	count := 0
	for iter := 0; count < targetDeals; iter++ {
		pc := playerCounts[iter%len(playerCounts)]
		r := rounds[(iter/len(playerCounts))%len(rounds)]
		if int(r)*pc > 51 {
			continue
		}
		count++

		moveMin := simulateMoveWithParam(modelID, dial.Param, dial.MinVal, pc, r, dial.IsBetting)
		moveMax := simulateMoveWithParam(modelID, dial.Param, dial.MaxVal, pc, r, dial.IsBetting)

		if moveMin != moveMax {
			switched++
			low := dial.MinVal
			high := dial.MaxVal
			for step := 0; step < 10; step++ {
				mid := (low + high) / 2
				mMid := simulateMoveWithParam(modelID, dial.Param, mid, pc, r, dial.IsBetting)
				if mMid == moveMin {
					low = mid
				} else {
					high = mid
				}
			}
			observedSwitches = append(observedSwitches, (low+high)/2)
		}
	}

	recMin := dial.MinVal
	recMax := dial.MaxVal
	anchor1 := 0.25
	anchor2 := 0.50
	anchor3 := 0.75

	if len(observedSwitches) > 0 {
		minObs := observedSwitches[0]
		maxObs := observedSwitches[0]
		for _, v := range observedSwitches {
			if v < minObs {
				minObs = v
			}
			if v > maxObs {
				maxObs = v
			}
		}
		span := maxObs - minObs
		if span > 0.05 {
			recMin = math.Max(dial.MinVal, minObs-span*0.08)
			recMax = math.Min(dial.MaxVal, maxObs+span*0.08)
		}
		anchor1 = (observedSwitches[0] - recMin) / (recMax - recMin)
		anchor2 = (observedSwitches[len(observedSwitches)/2] - recMin) / (recMax - recMin)
		anchor3 = (observedSwitches[len(observedSwitches)-1] - recMin) / (recMax - recMin)
	}

	return &ParamSummary{
		TotalTested:      count,
		SwitchedCount:    switched,
		SensitivityScore: float64(switched) / float64(count),
		RecommendedMin:   recMin,
		RecommendedMax:   recMax,
		KeyAnchor1UI:     clamp(anchor1, 0.05, 0.95),
		KeyAnchor2UI:     clamp(anchor2, 0.10, 0.90),
		KeyAnchor3UI:     clamp(anchor3, 0.15, 0.95),
	}
}

func simulateMoveWithParam(modelID uint8, param string, val float64, numPlayers int, round uint8, isBetting bool) int {
	gameBase := &game.GameBase{
		GameEndedChannel: make(chan string, 1),
	}
	f := fiddlesticks.InitializeGame(gameBase)

	params := map[string]string{
		param: strconv.FormatFloat(val, 'f', 4, 64),
	}
	f.AddInternalAiPlayer(modelID, params)

	for i := 1; i < numPlayers; i++ {
		f.AddInternalAiPlayer(3, nil)
	}

	f.SetSettings(round, round, 10, 3, 1)
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

func clamp(v, min, max float64) float64 {
	if v < min {
		return min
	}
	if v > max {
		return max
	}
	return v
}

func main() {
	models := []struct {
		ID   uint8
		Name string
	}{
		{ID: 5, Name: "Model 4a"},
		{ID: 6, Name: "Model 4b"},
		{ID: 7, Name: "Model 4c"},
	}

	reports := make(map[string]*ModelEvalReport)

	for _, m := range models {
		fmt.Printf("\n=== Running 105 Situation Sensitivity Bench for %s ===\n", m.Name)
		rep := &ModelEvalReport{
			Model:        m.Name,
			ParamResults: make(map[string]*ParamSummary),
		}

		for _, dial := range dialSpecs {
			if dial.Param == "bid_trust" && m.ID == 5 {
				continue
			}

			summary := evaluateDialSensitivity(m.ID, dial)
			rep.ParamResults[dial.Param] = summary
			rep.TotalSituations += summary.TotalTested
			rep.ResponsiveCount += summary.SwitchedCount

			fmt.Printf("  %-20s | Tested: %3d | Switched: %2d (%5.1f%%) | Rec Range: [%.2f, %.2f] | Anchors UI: [%.2f, %.2f, %.2f]\n",
				dial.Param, summary.TotalTested, summary.SwitchedCount, summary.SensitivityScore*100,
				summary.RecommendedMin, summary.RecommendedMax,
				summary.KeyAnchor1UI, summary.KeyAnchor2UI, summary.KeyAnchor3UI)
		}

		if rep.TotalSituations > 0 {
			rep.ResponsivenessPct = float64(rep.ResponsiveCount) / float64(rep.TotalSituations) * 100.0
		}
		reports[m.Name] = rep
		fmt.Printf(">> %s Total Responsiveness: %d/%d (%.1f%% of situations flipped by dialing params)\n",
			m.Name, rep.ResponsiveCount, rep.TotalSituations, rep.ResponsivenessPct)
	}

	_ = os.MkdirAll("outputs/behavior_bench", 0755)
	b, _ := json.MarshalIndent(reports, "", "  ")
	_ = os.WriteFile("outputs/behavior_bench/behavioral_report.json", b, 0644)
	fmt.Println("\nSaved full behavioral report to outputs/behavior_bench/behavioral_report.json")
}
