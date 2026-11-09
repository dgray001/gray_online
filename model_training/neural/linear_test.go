package neural

import (
	"math"
	"path/filepath"
	"testing"
)

func TestLinearGradientAndPersistence(t *testing.T) {
	n, updated := NewNetwork(2, 0, 3, 7), NewNetwork(2, 0, 3, 7)
	input, legal := []float64{0.3, -0.7}, []int{0, 2}
	updated.Train(input, legal, 2, 1)
	checkGradients(t, n, input, legal, [][][]float64{n.OutputWeights, {n.OutputBias}}, [][][]float64{updated.OutputWeights, {updated.OutputBias}})
	path := filepath.Join(t.TempDir(), "linear.json")
	if err := n.Save(path); err != nil {
		t.Fatal(err)
	}
	loaded, err := Load(path)
	if err != nil {
		t.Fatal(err)
	}
	if loaded.InputCount() != 2 || len(loaded.HiddenBias) != 0 || loaded.Predict(input, legal)[2] != n.Predict(input, legal)[2] {
		t.Fatal("linear model changed")
	}
}

func TestLinearLearnsSoftTargets(t *testing.T) {
	n := NewNetwork(2, 0, 3, 7)
	input, legal, targets := []float64{1, 0}, []int{0, 2}, []float64{0.25, 0.75}
	for i := 0; i < 1000; i++ {
		n.TrainDistribution(input, legal, targets, 0.1)
	}
	probabilities := n.Predict(input, legal)
	if math.Abs(probabilities[0]-0.25) > 1e-5 || probabilities[1] != 0 {
		t.Fatal(probabilities)
	}
}
