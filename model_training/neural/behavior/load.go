package behavior

import (
	"encoding/json"
	"fiddlesticks.live/neural"
	"fiddlesticks.live/neural/experiment"
	"fmt"
	"os"
	"path/filepath"
)

func loadModels(run string) ([]string, []*neural.Network, error) {
	data, err := os.ReadFile(filepath.Join(run, "trials.json"))
	if err != nil {
		return nil, nil, err
	}
	var results []experiment.Result
	if err := json.Unmarshal(data, &results); err != nil {
		return nil, nil, err
	}
	names, models := []string{}, []*neural.Network{}
	for _, result := range results {
		model, err := neural.Load(filepath.Join(run, filepath.Base(result.Model)))
		if err != nil {
			return nil, nil, err
		}
		if (model.InputCount() != neural.Inputs && model.InputCount() != neural.LegacyInputs) || len(model.OutputBias) != neural.Outputs {
			return nil, nil, fmt.Errorf("expected compatible checkpoint: %s", result.Model)
		}
		names, models = append(names, result.Trial.ID), append(models, model)
	}
	if len(models) < 2 {
		return nil, nil, fmt.Errorf("need at least two checkpoints")
	}
	return names, models, nil
}
