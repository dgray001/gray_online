package neural

func (n *Network) updateLinear(input []float64, legal []int, delta []float64, rate float64) {
	for _, action := range legal {
		n.OutputBias[action] -= rate * delta[action]
	}
	for j, value := range input {
		if value == 0 {
			continue
		}
		for _, action := range legal {
			n.OutputWeights[action][j] -= rate * delta[action] * value
		}
	}
}
