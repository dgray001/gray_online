package neural

import (
	"math"
	"path/filepath"
	"testing"
)

func TestBackpropagationMatchesFiniteDifferences(t *testing.T) {
	n := NewNetwork(2, 3, 3, 7)
	updated := NewNetwork(2, 3, 3, 7)
	input, legal := []float64{0.3, -0.7}, []int{0, 2}
	updated.Train(input, legal, 2, 1)
	groups := [][][]float64{n.HiddenWeights, n.OutputWeights, {n.HiddenBias}, {n.OutputBias}}
	trained := [][][]float64{updated.HiddenWeights, updated.OutputWeights, {updated.HiddenBias}, {updated.OutputBias}}
	checkGradients(t, n, input, legal, groups, trained)
}

func checkGradients(t *testing.T, n *Network, input []float64, legal []int, groups [][][]float64, trained [][][]float64) {
	t.Helper()
	for group, rows := range groups {
		for row, values := range rows {
			for column, value := range values {
				values[column] = value + 1e-5
				plus := -math.Log(n.Predict(input, legal)[2])
				values[column] = value - 1e-5
				minus := -math.Log(n.Predict(input, legal)[2])
				values[column] = value
				if math.Abs((plus-minus)/2e-5-(value-trained[group][row][column])) > 1e-6 {
					t.Fatalf("wrong gradient at %d/%d/%d", group, row, column)
				}
			}
		}
	}
}

func TestNetworkLearnsXOR(t *testing.T) {
	n := NewNetwork(2, 8, 3, 1)
	inputs := [][]float64{{0, 0}, {0, 1}, {1, 0}, {1, 1}}
	targets, legal := []int{0, 1, 1, 0}, []int{0, 1}
	for epoch := 0; epoch < 2000; epoch++ {
		for i, input := range inputs {
			n.Train(input, legal, targets[i], 0.1)
		}
	}
	for i, input := range inputs {
		probabilities := n.Predict(input, legal)
		if probabilities[targets[i]] < 0.98 || probabilities[2] != 0 {
			t.Fatal(probabilities)
		}
	}
	path := filepath.Join(t.TempDir(), "network.json")
	if err := n.Save(path); err != nil {
		t.Fatal(err)
	}
	loaded, err := Load(path)
	if err != nil {
		t.Fatal(err)
	}
	if loaded.Predict(inputs[0], legal)[0] != n.Predict(inputs[0], legal)[0] {
		t.Fatal("save/load changed predictions")
	}
}
