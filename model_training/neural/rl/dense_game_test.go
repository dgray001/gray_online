package rl

import (
	"reflect"
	"testing"
)

func TestHiddenPolicyLearnsFromFullGame(t *testing.T) {
	n := initialNetwork(configuredTrial(4, "xavier", 0.002, 0.01, 1))
	before := n.Clone()
	s := schedule(1, 52)[0]
	s.Settings.MaxRound = 2
	l := &Learner{Rate: 0.002, Entropy: 0.01, Baselines: map[Context]Baseline{}}
	episode := play(n, s, l, nil)
	if len(episode.Outcomes) != 3 || reflect.DeepEqual(n.HiddenWeights, before.HiddenWeights) {
		t.Fatal("hidden weights did not learn from outcomes")
	}
	if len(n.HiddenBias) != 4 || parameterCount(4) != 3359 {
		t.Fatal("wrong architecture")
	}
	before = n.Clone()
	play(n, s, nil, nil)
	if !reflect.DeepEqual(n, before) {
		t.Fatal("evaluation changed hidden policy")
	}
}
