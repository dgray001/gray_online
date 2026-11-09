package behavior

import (
	"fiddlesticks.live/neural"
	"math"
	"testing"
)

func TestBidComparisonUsesEffectiveEnginePolicy(t *testing.T) {
	a, b := neural.NewNetwork(1, 0, 79, 1), neural.NewNetwork(1, 0, 79, 1)
	for i, probability := range []float64{0.25, 0.5, 0.25} {
		a.OutputBias[i] = math.Log(probability)
	}
	for i, probability := range []float64{0.45, 0.1, 0.45} {
		b.OutputBias[i] = math.Log(probability)
	}
	example := neural.Example{Input: []float64{0}, Legal: []int{0, 1, 2}}
	x, y := predict(a, example), predict(b, example)
	if math.Abs(x.MeanBid-1) > 1e-12 || math.Abs(y.MeanBid-1) > 1e-12 {
		t.Fatal(x, y)
	}
	for i := range x.Probabilities {
		if math.Abs(x.Probabilities[i]-y.Probabilities[i]) > 1e-12 {
			t.Fatal("compared raw bid logits instead of engine behavior")
		}
	}
}

func TestCategoricalPolicyPreservesBidDistribution(t *testing.T) {
	n := neural.NewNetwork(1, 0, 79, 1)
	n.BidPolicy = "categorical"
	for i, p := range []float64{0.45, 0.1, 0.45} {
		n.OutputBias[i] = math.Log(p)
	}
	prediction := predict(n, neural.Example{Input: []float64{0}, Legal: []int{0, 1, 2}})
	if math.Abs(prediction.Probabilities[0]-0.45) > 1e-12 || math.Abs(prediction.MeanBid-1) > 1e-12 {
		t.Fatal("RL policy collapsed to expected bid", prediction)
	}
}
