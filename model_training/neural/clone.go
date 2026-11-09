package neural

import "slices"

func (n *Network) Clone() *Network {
	result := &Network{FeatureVersion: n.FeatureVersion, BidPolicy: n.BidPolicy, HiddenBias: slices.Clone(n.HiddenBias), OutputBias: slices.Clone(n.OutputBias)}
	for _, row := range n.HiddenWeights {
		result.HiddenWeights = append(result.HiddenWeights, slices.Clone(row))
	}
	for _, row := range n.OutputWeights {
		result.OutputWeights = append(result.OutputWeights, slices.Clone(row))
	}
	return result
}
