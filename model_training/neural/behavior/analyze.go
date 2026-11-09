package behavior

import (
	"fiddlesticks.live/neural"
	"fiddlesticks.live/neural/experiment"
)

func analyze(names []string, models []*neural.Network, set experiment.StandardSet) (Summary, [][]Prediction) {
	profiles := make([][]Prediction, len(models))
	for i, model := range models {
		profiles[i] = make([]Prediction, len(set.Situations))
		for j, example := range set.Situations {
			profiles[i][j] = predict(model, example)
		}
	}
	summary := Summary{Trials: names, Games: len(set.Games), Situations: len(set.Situations)}
	for i := range models {
		for j := i + 1; j < len(models); j++ {
			pair := compare(names[i], names[j], profiles[i], profiles[j], set.Situations)
			if len(models[i].HiddenBias) == len(models[j].HiddenBias) {
				pair.ParameterRMS = parameterRMS(models[i], models[j])
				pair.ParameterComparable = true
			}
			summary.Pairs = append(summary.Pairs, pair)
		}
	}
	return summary, profiles
}
