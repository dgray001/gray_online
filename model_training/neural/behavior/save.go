package behavior

import (
	"encoding/json"
	"fiddlesticks.live/neural/experiment"
	"os"
	"path/filepath"
)

func save(options Options, set experiment.StandardSet, summary Summary, profiles [][]Prediction) error {
	files := []struct {
		Name  string
		Value any
	}{{"options.json", options}, {"corpus.json", set}, {"summary.json", summary}, {"fingerprints.json", struct {
		Trials      []string
		Predictions [][]Prediction
	}{summary.Trials, profiles}}}
	for _, file := range files {
		data, err := json.Marshal(file.Value)
		if err != nil {
			return err
		}
		if err := os.WriteFile(filepath.Join(options.Output, file.Name), data, 0600); err != nil {
			return err
		}
	}
	if err := matrix(options.Output, summary); err != nil {
		return err
	}
	return report(options.Output, summary)
}
