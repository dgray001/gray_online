package rl

import (
	"fiddlesticks.live/neural"
	"math"
	"testing"
)

func TestPolicyAuditDistinguishesUniformAndDeterministicChoices(t *testing.T) {
	audit := PolicyAudit{}
	p := make([]float64, neural.Outputs)
	p[27], p[28] = 0.5, 0.5
	audit.observe([]int{27, 28}, p)
	p[27], p[28] = 1, 0
	audit.observe([]int{27, 28}, p)
	m := audit.averaged().Cards
	if math.Abs(m.NormalizedEntropy-0.5) > 1e-12 || m.UniformTV != 0.25 || m.MaxProbability != 0.75 {
		t.Fatal(m)
	}
}
