package behavior

import (
	"encoding/json"
	"fiddlesticks.live/neural/experiment"
	"os"
)

func corpus(options Options) (experiment.StandardSet, error) {
	if options.Corpus == "" {
		return experiment.StandardSituations(options.Games, options.Seed), nil
	}
	data, err := os.ReadFile(options.Corpus)
	if err != nil {
		return experiment.StandardSet{}, err
	}
	var result experiment.StandardSet
	err = json.Unmarshal(data, &result)
	return result, err
}
