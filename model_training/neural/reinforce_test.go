package neural

import (
	"math"
	"testing"
)

func TestReinforceGradient(t *testing.T) {
	n := NewNetwork(2, 0, 3, 7)
	updated := n.Clone()
	input, legal := []float64{0.3, -0.7}, []int{0, 2}
	p := n.Predict(input, legal)
	step := PolicyStep{Input: input, Legal: legal, Action: 2, Probabilities: []float64{p[0], p[2]}}
	updated.ReinforceLinear([]PolicyStep{step}, 0.7, 1, 0.02)
	groups, trained := [][][]float64{n.OutputWeights, {n.OutputBias}}, [][][]float64{updated.OutputWeights, {updated.OutputBias}}
	checkPolicyGradients(t, n, step, groups, trained)
}

func policyLoss(n *Network, step PolicyStep) float64 {
	p := n.Predict(step.Input, step.Legal)
	loss := -0.7 * math.Log(p[step.Action])
	for _, action := range step.Legal {
		loss += 0.02 * p[action] * math.Log(p[action])
	}
	return loss
}
