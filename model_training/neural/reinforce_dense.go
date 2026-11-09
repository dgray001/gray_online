package neural

func (n *Network) Reinforce(steps []PolicyStep, advantage float64, rate float64, entropy float64) {
	if len(n.HiddenBias) == 0 {
		n.ReinforceLinear(steps, advantage, rate, entropy)
		return
	}
	frozen := n.outputSnapshot()
	activations := make([][]float64, len(steps))
	for i, step := range steps {
		activations[i] = step.Hidden
		if activations[i] == nil {
			activations[i], _ = n.forwardLegal(step.Input, step.Legal)
		}
	}
	for i, step := range steps {
		hidden := activations[i]
		delta := policyDelta(len(n.OutputBias), step, advantage, entropy)
		hiddenDelta := make([]float64, len(hidden))
		for _, action := range step.Legal {
			for j, weight := range frozen[action] {
				hiddenDelta[j] += delta[action] * weight * (1 - hidden[j]*hidden[j])
			}
		}
		n.updateWeights(step.Input, hidden, step.Legal, delta, hiddenDelta, rate)
	}
}
