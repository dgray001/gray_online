package rl

import (
	"reflect"
	"testing"
)

func TestGameTrainsOnOutcomesAndEvaluationIsReadOnly(t *testing.T) {
	n := initialNetwork(trial("zero", 0.002, 0.01, 1))
	before := n.Clone()
	scenario := schedule(1, 901)[0]
	scenario.Settings.MaxRound = 2
	l := &Learner{Rate: 0.002, Entropy: 0.01, Baselines: map[Context]Baseline{}}
	episode := play(n, scenario, l, nil)
	count, successes := 0, 0.0
	for _, baseline := range l.Baselines {
		count += baseline.Count
		successes += baseline.Successes
	}
	if len(episode.Outcomes) != 3 || count != 3 || reflect.DeepEqual(n, before) {
		t.Fatal("round outcomes did not train policy")
	}
	metrics := summarize("test", []Episode{episode}).All
	if float64(metrics.Exact) != successes {
		t.Fatal("reward differs from exact-bid outcomes")
	}
	before = n.Clone()
	play(n, scenario, nil, nil)
	if !reflect.DeepEqual(n, before) {
		t.Fatal("evaluation changed policy")
	}
}
