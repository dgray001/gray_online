package neural

import (
	"math"
	"testing"
)

func reinforceReference(n *Network, steps []PolicyStep) {
	frozen := n.Clone()
	for _, step := range steps {
		hidden, _ := frozen.forward(step.Input)
		delta := policyDelta(len(n.OutputBias), step, 0.7, 0.02)
		hd := make([]float64, len(hidden))
		for _, action := range step.Legal {
			for j, weight := range frozen.OutputWeights[action] {
				hd[j] += delta[action] * weight * (1 - hidden[j]*hidden[j])
			}
		}
		n.updateWeights(step.Input, hidden, step.Legal, delta, hd, 0.003)
	}
}

func TestCachedRoundGradientMatchesReference(t *testing.T) {
	for _, width := range []int{8, 16, 32, 64, 128, 256} {
		n := NewNetwork(Inputs, width, Outputs, 7)
		reference := n.Clone()
		steps := []PolicyStep{}
		for k := 0; k < 3; k++ {
			input := make([]float64, Inputs)
			for i := k; i < len(input); i += 9 {
				input[i] = 0.25
			}
			legal := []int{0, 1, 2}
			p, hidden := n.PredictWithHidden(input, legal)
			steps = append(steps, PolicyStep{Input: input, Legal: legal, Action: k, Probabilities: []float64{p[0], p[1], p[2]}, Hidden: hidden})
		}
		n.Reinforce(steps, 0.7, 0.003, 0.02)
		reinforceReference(reference, steps)
		actual := [][][]float64{n.HiddenWeights, n.OutputWeights, {n.HiddenBias}, {n.OutputBias}}
		expected := [][][]float64{reference.HiddenWeights, reference.OutputWeights, {reference.HiddenBias}, {reference.OutputBias}}
		for group, rows := range actual {
			for i, row := range rows {
				for j, value := range row {
					if math.Abs(value-expected[group][i][j]) > 1e-12 {
						t.Fatalf("gradient mismatch width %d", width)
					}
				}
			}
		}
	}
}
