package rl

import (
	"math"
	"testing"
)

func TestAuditMatchesSignedWeightSampler(t *testing.T) {
	cases := []struct {
		Weights       []float64
		Probabilities []float64
	}{
		{[]float64{0.25, 0.75}, []float64{0.25, 0.75}},
		{[]float64{0.8, -0.4, 0.6}, []float64{0.8, 0, 0.2}},
		{[]float64{-1, -1}, []float64{1, 0}},
		{[]float64{0, 1, -1}, []float64{0, 1, 0}},
		{[]float64{0, 0}, []float64{1, 0}},
	}
	for _, test := range cases {
		got := samplerProbabilities(test.Weights)
		for i, p := range got {
			if math.Abs(p-test.Probabilities[i]) > 1e-12 {
				t.Fatal(test, got)
			}
		}
	}
}
