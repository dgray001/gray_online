package neural

import "math"

func (n *Network) Train(input []float64, legal []int, target int, rate float64) float64 {
	if len(n.HiddenBias) == 0 {
		targets := make([]float64, len(legal))
		for i, action := range legal {
			if action == target {
				targets[i] = 1
			}
		}
		return n.TrainDistribution(input, legal, targets, rate)
	}
	hidden, logits := n.forward(input)
	delta := maskedSoftmax(logits, legal)
	loss := -math.Log(math.Max(delta[target], 1e-300))
	delta[target]--
	n.backpropagate(input, hidden, legal, delta, rate)
	return loss
}
