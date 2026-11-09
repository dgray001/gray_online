package neural

import "fmt"

func (n *Network) validate() error {
	if len(n.OutputBias) == 0 || len(n.HiddenWeights) != len(n.HiddenBias) || len(n.OutputWeights) != len(n.OutputBias) {
		return fmt.Errorf("invalid network dimensions")
	}
	inputs := n.InputCount()
	if inputs == 0 {
		return fmt.Errorf("network has no inputs")
	}
	for _, weights := range n.HiddenWeights {
		if len(weights) != inputs {
			return fmt.Errorf("inconsistent hidden layer dimensions")
		}
	}
	for _, weights := range n.OutputWeights {
		columns := len(n.HiddenBias)
		if columns == 0 {
			columns = inputs
		}
		if len(weights) != columns {
			return fmt.Errorf("inconsistent output layer dimensions")
		}
	}
	return nil
}

func (n *Network) InputCount() int {
	if len(n.HiddenWeights) == 0 {
		return len(n.OutputWeights[0])
	}
	return len(n.HiddenWeights[0])
}
