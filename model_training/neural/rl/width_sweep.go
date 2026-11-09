package rl

func widthSweep(options Options) []Trial {
	result := []Trial{}
	for _, width := range options.Widths {
		configuration := options
		configuration.Widths, configuration.Hidden, configuration.Sweep, configuration.LinearControls = nil, width, true, false
		result = append(result, trials(configuration)...)
	}
	if options.LinearControls {
		for _, seed := range []int64{1, 2, 3} {
			result = append(result, configuredTrial(0, "xavier", 0.002, 0, seed))
		}
	}
	return result
}
