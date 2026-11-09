package neural

import (
	"math"
	"math/rand"
)

type Network struct {
	FeatureVersion int    `json:",omitempty"`
	BidPolicy      string `json:",omitempty"`
	HiddenWeights  [][]float64
	HiddenBias     []float64
	OutputWeights  [][]float64
	OutputBias     []float64
}

func NewNetwork(inputs int, hidden int, outputs int, seed int64) *Network {
	rng := rand.New(rand.NewSource(seed))
	if hidden == 0 {
		return &Network{OutputWeights: randomWeights(outputs, inputs, rng), OutputBias: make([]float64, outputs)}
	}
	return &Network{HiddenWeights: randomWeights(hidden, inputs, rng), HiddenBias: make([]float64, hidden), OutputWeights: randomWeights(outputs, hidden, rng), OutputBias: make([]float64, outputs)}
}

func randomWeights(rows int, columns int, rng *rand.Rand) [][]float64 {
	weights := make([][]float64, rows)
	limit := math.Sqrt(6 / float64(rows+columns))
	for row := range weights {
		weights[row] = make([]float64, columns)
		for column := range weights[row] {
			weights[row][column] = (2*rng.Float64() - 1) * limit
		}
	}
	return weights
}

func (n *Network) forward(input []float64) ([]float64, []float64) {
	hidden := make([]float64, len(n.HiddenBias))
	for i, weights := range n.HiddenWeights {
		value := n.HiddenBias[i]
		for j, weight := range weights {
			value += weight * input[j]
		}
		hidden[i] = math.Tanh(value)
	}
	logits := make([]float64, len(n.OutputBias))
	if len(n.HiddenBias) == 0 {
		hidden = input
	}
	for i, weights := range n.OutputWeights {
		logits[i] = n.OutputBias[i]
		for j, weight := range weights {
			logits[i] += weight * hidden[j]
		}
	}
	return hidden, logits
}
