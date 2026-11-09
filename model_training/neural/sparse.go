package neural

func nonzeroIndices(input []float64) []int {
	indices := make([]int, 0, len(input))
	for i, value := range input {
		if value != 0 {
			indices = append(indices, i)
		}
	}
	return indices
}
