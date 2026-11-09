package neural

func (n *Network) linearLogits(input []float64, legal []int) []float64 {
	logits := make([]float64, len(n.OutputBias))
	for _, action := range legal {
		logits[action] = n.OutputBias[action]
	}
	for j, value := range input {
		if value == 0 {
			continue
		}
		for _, action := range legal {
			logits[action] += value * n.OutputWeights[action][j]
		}
	}
	return logits
}
