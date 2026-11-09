package experiment

import (
	"fiddlesticks.live/neural"
	"math"
)

func predictionMetrics(probabilities []float64, example neural.Example, method string) (float64, float64, bool) {
	kl, predicted := 0.0, example.Legal[0]
	for i, action := range example.Legal {
		if probabilities[action] > probabilities[predicted] {
			predicted = action
		}
		if target := example.Distribution[i]; target > 0 {
			kl += target * math.Log(target/math.Max(probabilities[action], 1e-300))
		}
	}
	loss := -math.Log(math.Max(probabilities[example.Target], 1e-300))
	if method == "soft" {
		loss = kl
	}
	return loss, kl, predicted == example.Target
}
