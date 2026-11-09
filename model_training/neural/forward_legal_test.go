package neural

import (
	"math"
	"testing"
)

func TestSparseForwardMatchesDenseAtEveryWidth(t *testing.T) {
	input := make([]float64, Inputs)
	for i := 0; i < len(input); i += 7 {
		input[i] = float64(i%11) / 11
	}
	legal := []int{27, 40, 52, 78}
	for _, width := range []int{4, 8, 16, 32, 64, 128, 256} {
		n := NewNetwork(Inputs, width, Outputs, 7)
		hidden, logits := n.forward(input)
		sparseHidden, sparseLogits := n.forwardLegal(input, legal)
		for i, value := range hidden {
			if math.Abs(value-sparseHidden[i]) > 1e-12 {
				t.Fatalf("hidden mismatch width %d", width)
			}
		}
		for _, action := range legal {
			if math.Abs(logits[action]-sparseLogits[action]) > 1e-12 {
				t.Fatalf("output mismatch width %d", width)
			}
		}
	}
}
