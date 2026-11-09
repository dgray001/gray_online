package behavior

import (
	"fiddlesticks.live/neural"
	"math"
	"testing"
)

func TestCommonWeightShiftDoesNotChangeBehavior(t *testing.T) {
	a := neural.NewNetwork(2, 0, 79, 7)
	b := a.Clone()
	for i := range b.OutputWeights {
		b.OutputWeights[i][0] += 1.5
		b.OutputWeights[i][1] -= 0.3
		b.OutputBias[i] += 0.4
	}
	examples := []neural.Example{{Input: []float64{0.3, 0.8}, Legal: []int{27, 40, 78}}, {Input: []float64{0.2, 0.1}, Legal: []int{0, 1, 2, 3}}}
	x, y := []Prediction{}, []Prediction{}
	for _, example := range examples {
		x = append(x, predict(a, example))
		y = append(y, predict(b, example))
	}
	pair := compare("a", "b", x, y, examples)
	if pair.CardTV > 1e-12 || pair.BidMAE > 1e-12 || parameterRMS(a, b) == 0 {
		t.Fatal(pair)
	}
}

func TestCardDistributionsDifferDespiteSameTopChoice(t *testing.T) {
	a, b := neural.NewNetwork(1, 0, 79, 1), neural.NewNetwork(1, 0, 79, 1)
	a.OutputBias[27], a.OutputBias[28] = math.Log(0.6), math.Log(0.4)
	b.OutputBias[27], b.OutputBias[28] = math.Log(0.9), math.Log(0.1)
	example := neural.Example{Input: []float64{0}, Legal: []int{27, 28}}
	pair := compare("a", "b", []Prediction{predict(a, example)}, []Prediction{predict(b, example)}, []neural.Example{example})
	if math.Abs(pair.CardTV-0.3) > 1e-12 || pair.CardDisagreement != 0 {
		t.Fatal(pair)
	}
}
