package rl

import "fiddlesticks.live/neural"

func testModels(selected []Result, options Options) ([]string, []*neural.Network, error) {
	names := []string{"theory4", "untrained-zero", "untrained-xavier"}
	models := []*neural.Network{nil, initialNetwork(trial("zero", 0.002, 0, 1)), initialNetwork(trial("xavier", 0.002, 0, 1))}
	if options.Reference != "" {
		network, err := neural.Load(options.Reference)
		if err != nil {
			return nil, nil, err
		}
		names, models = append(names, "previous-linear"), append(models, network)
	}
	if options.ReferenceHidden4 != "" {
		network, err := neural.Load(options.ReferenceHidden4)
		if err != nil {
			return nil, nil, err
		}
		names, models = append(names, "previous-hidden4"), append(models, network)
	}
	for _, result := range selected {
		network, err := neural.Load(result.Model)
		if err != nil {
			return nil, nil, err
		}
		names, models = append(names, result.Trial.ID), append(models, network)
	}
	return names, models, nil
}
