package experiment

import (
	"fmt"
)

type Regimen struct {
	ID        string
	Seeds     int
	Margin    float64
	MarginSD  float64
	WinShare  float64
	TeacherKL float64
	Accuracy  float64
}

func regimenID(trial Trial) string {
	return fmt.Sprintf("%s-%s-%.2f", trial.Method, trial.Start, trial.Rate)
}

func regimens(results []Result, evaluations []Evaluation) []Regimen {
	byAgent, groups := map[string]Evaluation{}, map[string]*Regimen{}
	for _, evaluation := range evaluations {
		if evaluation.Group == "all" {
			byAgent[evaluation.Agent] = evaluation
		}
	}
	for _, result := range results {
		id, m := regimenID(result.Trial), byAgent[result.Trial.ID]
		if groups[id] == nil {
			groups[id] = &Regimen{ID: id}
		}
		g := groups[id]
		g.Seeds++
		g.Margin += m.Margin
		g.MarginSD += m.Margin * m.Margin
		g.WinShare += m.WinShare
		g.TeacherKL += result.Validation.TeacherKL
		g.Accuracy += result.Validation.Accuracy
	}
	return finishRegimens(groups)
}
