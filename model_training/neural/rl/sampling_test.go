package rl

import (
	"fiddlesticks.live/neural"
	"math"
	"math/rand"
	"testing"
)

func TestBidsSampleCategoricalPolicy(t *testing.T) {
	n := initialNetwork(trial("zero", 0.002, 0, 1))
	for i, p := range []float64{0.45, 0.1, 0.45} {
		n.OutputBias[i] = math.Log(p)
	}
	n.OutputBias[50] = 100
	agent := Agent{Network: n, Random: rand.New(rand.NewSource(1))}
	counts := [3]int{}
	for i := 0; i < 500; i++ {
		action := agent.choose(make([]float64, neural.Inputs), []int{0, 1, 2})
		counts[action]++
	}
	if counts[0] < 150 || counts[2] < 150 || counts[1] > 100 {
		t.Fatal("used rounded expected bid instead of categorical sampling", counts)
	}
}
