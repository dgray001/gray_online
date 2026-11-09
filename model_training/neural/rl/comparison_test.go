package rl

import (
	"reflect"
	"testing"
)

func TestAuditingDoesNotChangeNeuralGame(t *testing.T) {
	n := initialNetwork(configuredTrial(8, "xavier", 0.002, 0, 1))
	s := schedule(1, 903)[0]
	s.Settings.Players, s.Settings.MaxRound = 4, 2
	expected := play(n, s, nil, nil)
	audit := PolicyAudit{}
	actual := comparisonGame(benchmarkAgent{Name: "test", Model: 4, Network: n}, s, &audit)
	if !reflect.DeepEqual(actual, expected) || audit.Bids.Decisions != 3 {
		t.Fatal("instrumentation changed game or failed to record choices")
	}
}

func TestNativeRandomModelIsReproducible(t *testing.T) {
	s := schedule(1, 904)[0]
	s.Settings.Players, s.Settings.MaxRound = 4, 2
	a, b := PolicyAudit{}, PolicyAudit{}
	if !reflect.DeepEqual(comparisonGame(benchmarkAgent{Model: 0}, s, &a), comparisonGame(benchmarkAgent{Model: 0}, s, &b)) {
		t.Fatal("native random model is not seeded")
	}
}
