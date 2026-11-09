package rl

import "fiddlesticks.live/neural"

func parameterCount(hidden int) int {
	return parameterCountFor(neural.Inputs, hidden)
}

func parameterCountFor(inputs int, hidden int) int {
	if inputs == 0 {
		inputs = neural.LegacyInputs
	}
	if hidden == 0 {
		return (inputs + 1) * neural.Outputs
	}
	return (inputs+1)*hidden + (hidden+1)*neural.Outputs
}
