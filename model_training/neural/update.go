package neural

func (n *Network) updateWeights(input []float64, hidden []float64, legal []int, delta []float64, hiddenDelta []float64, rate float64) {
	active := nonzeroIndices(input)
	for _, action := range legal {
		n.OutputBias[action] -= rate * delta[action]
		for j := range hidden {
			n.OutputWeights[action][j] -= rate * delta[action] * hidden[j]
		}
	}
	for i := range hidden {
		n.HiddenBias[i] -= rate * hiddenDelta[i]
		for _, j := range active {
			n.HiddenWeights[i][j] -= rate * hiddenDelta[i] * input[j]
		}
	}
}
