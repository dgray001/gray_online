package rl

import (
	"strings"
)

type Regimen struct {
	Inputs   int
	Hidden   int
	ID       string
	Seeds    int
	Initial  float64
	Exact    float64
	ExactSD  float64
	WinShare float64
	Margin   float64
}

func regimenID(trial Trial) string { return trial.ID[:strings.LastIndex(trial.ID, "-s")] }

func regimens(results []Result) []Regimen {
	groups := map[string]*Regimen{}
	for _, result := range results {
		id, m := regimenID(result.Trial), selectionMetrics(result.Validation)
		if groups[id] == nil {
			groups[id] = &Regimen{ID: id, Hidden: result.Trial.Hidden, Inputs: result.Trial.Inputs}
		}
		g := groups[id]
		g.Seeds++
		g.Initial += result.InitialCore.ExactRate
		g.Exact += m.ExactRate
		g.ExactSD += m.ExactRate * m.ExactRate
		g.WinShare += m.WinShare
		g.Margin += m.Margin
	}
	return finishRegimens(groups)
}
