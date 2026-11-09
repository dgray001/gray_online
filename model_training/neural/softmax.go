package neural

import "math"

func maskedSoftmax(logits []float64, legal []int) []float64 {
	maximum := math.Inf(-1)
	for _, action := range legal {
		maximum = math.Max(maximum, logits[action])
	}
	probabilities := make([]float64, len(logits))
	total := 0.0
	for _, action := range legal {
		probabilities[action] = math.Exp(logits[action] - maximum)
		total += probabilities[action]
	}
	for _, action := range legal {
		probabilities[action] /= total
	}
	return probabilities
}

func (n *Network) Predict(input []float64, legal []int) []float64 {
	probabilities, _ := n.PredictWithHidden(input, legal)
	return probabilities
}

func (n *Network) PredictWithHidden(input []float64, legal []int) ([]float64, []float64) {
	if len(n.HiddenBias) == 0 {
		return maskedSoftmax(n.linearLogits(input, legal), legal), nil
	}
	hidden, logits := n.forwardLegal(input, legal)
	return maskedSoftmax(logits, legal), hidden
}
