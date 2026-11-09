package neural

func (n *Network) outputSnapshot() [][]float64 {
	columns := len(n.HiddenBias)
	values := make([]float64, len(n.OutputWeights)*columns)
	rows := make([][]float64, len(n.OutputWeights))
	for i, row := range n.OutputWeights {
		rows[i] = values[i*columns : (i+1)*columns]
		copy(rows[i], row)
	}
	return rows
}
