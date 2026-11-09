package experiment

import (
	"fiddlesticks.live/neural"
	"testing"
)

func TestEarlyStoppingRestoresBestCheckpoint(t *testing.T) {
	example := neural.Example{Input: make([]float64, neural.Inputs), Legal: []int{0, 1}, Target: 0, Distribution: []float64{0.5, 0.5}}
	data := []neural.Example{example}
	result, network := fit(Trial{Method: "soft", Start: "zero", Rate: 0.01, Seed: 1}, data, data, Options{MaxEpochs: 10, Patience: 2})
	if result.Stop != "validation plateau" || result.Epochs != 2 || result.BestEpoch != 0 {
		t.Fatal(result)
	}
	if network.Predict(example.Input, example.Legal)[0] != 0.5 {
		t.Fatal("best checkpoint lost")
	}
	example.Distribution = []float64{0, 1}
	example.Target = 1
	result, network = fit(Trial{Method: "hard", Start: "zero", Rate: 0.1, Seed: 1}, data, []neural.Example{example}, Options{MaxEpochs: 10, Patience: 2})
	if result.BestEpoch != 0 || network.OutputBias[0] != 0 {
		t.Fatal("validation selected a worse checkpoint")
	}
}
