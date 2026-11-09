package neural

import "math"

func (n *Network) forwardLegal(input []float64, legal []int) ([]float64, []float64) {
	active := nonzeroIndices(input)
	hidden := make([]float64, len(n.HiddenBias))
	for i, weights := range n.HiddenWeights {
		value := n.HiddenBias[i]
		for _, j := range active {
			value += weights[j] * input[j]
		}
		hidden[i] = math.Tanh(value)
	}
	logits := make([]float64, len(n.OutputBias))
	for _, action := range legal {
		logits[action] = n.OutputBias[action]
		for j, value := range hidden {
			logits[action] += n.OutputWeights[action][j] * value
		}
	}
	return hidden, logits
}
