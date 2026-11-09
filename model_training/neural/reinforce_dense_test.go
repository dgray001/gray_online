package neural

import (
	"math"
	"testing"
)

func TestDenseReinforceUsesFrozenTrajectoryGradients(t *testing.T) {
	n := NewNetwork(2, 4, 3, 7)
	updated := n.Clone()
	steps := []PolicyStep{}
	for i, input := range [][]float64{{0.3, -0.7}, {-0.2, 0.5}} {
		legal := []int{0, 2}
		p := n.Predict(input, legal)
		steps = append(steps, PolicyStep{Input: input, Legal: legal, Action: legal[i], Probabilities: []float64{p[0], p[2]}})
	}
	updated.Reinforce(steps, 0.7, 1, 0.02)
	groups := [][][]float64{n.HiddenWeights, n.OutputWeights, {n.HiddenBias}, {n.OutputBias}}
	trained := [][][]float64{updated.HiddenWeights, updated.OutputWeights, {updated.HiddenBias}, {updated.OutputBias}}
	loss := func() float64 {
		value := 0.0
		for _, step := range steps {
			value += policyLoss(n, step)
		}
		return value
	}
	for group, rows := range groups {
		for row, values := range rows {
			for column, value := range values {
				values[column] = value + 1e-5
				plus := loss()
				values[column] = value - 1e-5
				minus := loss()
				values[column] = value
				if math.Abs((plus-minus)/2e-5-(value-trained[group][row][column])) > 1e-6 {
					t.Fatalf("wrong frozen gradient %d/%d/%d", group, row, column)
				}
			}
		}
	}
}
