package neural

func (n *Network) backpropagate(input []float64, hidden []float64, legal []int, delta []float64, rate float64) {
	hiddenDelta := make([]float64, len(hidden))
	for _, action := range legal {
		for j, weight := range n.OutputWeights[action] {
			hiddenDelta[j] += delta[action] * weight * (1 - hidden[j]*hidden[j])
		}
	}
	n.updateWeights(input, hidden, legal, delta, hiddenDelta, rate)
}
