package neural

type PolicyStep struct {
	Hidden        []float64
	Input         []float64
	Legal         []int
	Action        int
	Probabilities []float64
}

func (n *Network) ReinforceLinear(steps []PolicyStep, advantage float64, rate float64, entropy float64) {
	for _, step := range steps {
		delta := policyDelta(len(n.OutputBias), step, advantage, entropy)
		n.updateLinear(step.Input, step.Legal, delta, rate)
	}
}
