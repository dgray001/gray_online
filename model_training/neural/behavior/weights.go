package behavior

import (
	"fiddlesticks.live/neural"
	"math"
)

func parameterRMS(a *neural.Network, b *neural.Network) float64 {
	sum, count := 0.0, 0
	for i, row := range a.HiddenWeights {
		for j, weight := range row {
			difference := weight - b.HiddenWeights[i][j]
			sum += difference * difference
			count++
		}
		difference := a.HiddenBias[i] - b.HiddenBias[i]
		sum += difference * difference
		count++
	}
	for i, row := range a.OutputWeights {
		for j, weight := range row {
			difference := weight - b.OutputWeights[i][j]
			sum += difference * difference
			count++
		}
		difference := a.OutputBias[i] - b.OutputBias[i]
		sum += difference * difference
		count++
	}
	return math.Sqrt(sum / float64(count))
}
