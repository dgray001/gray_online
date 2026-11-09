package neural

import "math"

func policyDelta(outputs int, step PolicyStep, advantage float64, entropy float64) []float64 {
	h := 0.0
	for _, probability := range step.Probabilities {
		if probability > 0 {
			h -= probability * math.Log(probability)
		}
	}
	delta := make([]float64, outputs)
	for i, action := range step.Legal {
		probability := step.Probabilities[i]
		delta[action] = advantage*probability + entropy*probability*(math.Log(math.Max(probability, 1e-300))+h)
		if action == step.Action {
			delta[action] -= advantage
		}
	}
	return delta
}
