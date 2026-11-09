package neural

import (
	"slices"
	"testing"
)

func TestNetworkCollectsAndPlaysCompleteGame(t *testing.T) {
	network := NewNetwork(Inputs, 8, Outputs, 1)
	examples := []Example{}
	settings := GameSettings{Players: 4, MaxRound: 3, RoundPoints: 7, TrickPoints: 0}
	if scores := Play(network, &examples, settings, 0); len(scores) != 4 {
		t.Fatal(scores)
	}
	if len(examples) == 0 {
		t.Fatal("no training examples")
	}
	for _, example := range examples {
		if example.Input[161] != 4.0/51 || example.Input[368] != 7.0/99 || example.Input[369] != 0 {
			t.Fatal("game settings missing from inputs")
		}
		if len(example.Input) != Inputs || !slices.Contains(example.Legal, example.Target) {
			t.Fatal("invalid training example")
		}
	}
	scores := Play(network, nil, settings, 0)
	if len(scores) != 4 {
		t.Fatal(scores)
	}
	for _, score := range scores {
		if score%7 != 0 {
			t.Fatal("game ignored scoring settings", scores)
		}
	}
}
