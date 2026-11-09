package behavior

import (
	"fiddlesticks.live/neural"
	"fiddlesticks.live/neural/experiment"
	"testing"
)

func TestArchitectureComparisonUsesBehaviorWithoutWeightDistance(t *testing.T) {
	a, b := neural.NewNetwork(2, 0, 3, 1), neural.NewNetwork(2, 4, 3, 1)
	a.BidPolicy, b.BidPolicy = "categorical", "categorical"
	set := experiment.StandardSet{Situations: []neural.Example{{Input: []float64{0.2, -0.3}, Legal: []int{0, 1}}}}
	summary, _ := analyze([]string{"a", "b"}, []*neural.Network{a, b}, set)
	if len(summary.Pairs) != 1 || summary.Pairs[0].ParameterComparable || weightText(summary.Pairs[0]) != "n/a" {
		t.Fatal(summary)
	}
	other := b.Clone()
	other.HiddenWeights[0][0]++
	if parameterRMS(b, other) == 0 {
		t.Fatal("weight comparison ignored hidden parameters")
	}
}
