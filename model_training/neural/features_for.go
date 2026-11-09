package neural

import "github.com/dgray001/gray_online/game/games/fiddlesticks"

func FeaturesFor(network *Network, state fiddlesticks.LearningState, betting bool) []float64 {
	if network != nil && network.InputCount() == LegacyInputs {
		return legacyFeatures(state, betting)
	}
	return Features(state, betting)
}
