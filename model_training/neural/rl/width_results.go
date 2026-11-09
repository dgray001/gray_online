package rl

import "slices"

type WidthResult struct {
	CorePoints        float64
	CoreScore         float64
	CoreOpponentScore float64
	CoreMargin        float64
	HistoricalPoints  float64
	HistoricalScore   float64
	Hidden            int
	Parameters        int
	Recipe            string
	Validation        float64
	SeedSD            float64
	CoreExact         float64
	CoreUpper         float64
	CoreWin           float64
	ZeroBids          float64
	Historical        float64
}

func widthResults(ranking []Regimen, mixed []Evaluation, historical []Evaluation) []WidthResult {
	seen, result := map[int]bool{}, []WidthResult{}
	for _, r := range ranking {
		if seen[r.Hidden] {
			continue
		}
		seen[r.Hidden] = true
		row := WidthResult{Hidden: r.Hidden, Parameters: parameterCountFor(r.Inputs, r.Hidden), Recipe: r.ID, Validation: r.Exact, SeedSD: r.ExactSD}
		for _, e := range mixed {
			if e.Name == r.ID+"-s1" {
				m := selectionMetrics(e)
				row.CoreExact, row.CoreUpper, row.CoreWin, row.ZeroBids = m.ExactRate, e.CoreUpper.ExactRate, m.WinShare, m.ZeroBidRate
				row.CorePoints, row.CoreScore, row.CoreOpponentScore, row.CoreMargin = m.Points, m.Score, m.OpponentScore, m.Margin
			}
		}
		for _, e := range historical {
			if e.Name == r.ID+"-s1" {
				row.Historical = e.All.ExactRate
				row.HistoricalPoints, row.HistoricalScore = e.All.Points, e.All.Score
			}
		}
		result = append(result, row)
	}
	slices.SortFunc(result, func(a WidthResult, b WidthResult) int { return a.Hidden - b.Hidden })
	return result
}
