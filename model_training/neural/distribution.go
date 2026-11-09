package neural

import "math"

func (n *Network) TrainDistribution(input []float64, legal []int, targets []float64, rate float64) float64 {
	var hidden, logits []float64
	if len(n.HiddenBias) == 0 {
		logits = n.linearLogits(input, legal)
	} else {
		hidden, logits = n.forward(input)
	}
	delta := maskedSoftmax(logits, legal)
	loss := 0.0
	for i, action := range legal {
		loss -= targets[i] * math.Log(math.Max(delta[action], 1e-300))
		delta[action] -= targets[i]
	}
	if len(n.HiddenBias) == 0 {
		n.updateLinear(input, legal, delta, rate)
	} else {
		n.backpropagate(input, hidden, legal, delta, rate)
	}
	return loss
}
