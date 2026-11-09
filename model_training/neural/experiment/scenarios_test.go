package experiment

import (
	"reflect"
	"testing"
)

func TestScenarioCoverageAndSplitIsolation(t *testing.T) {
	a, b := scenarios(400, 11), scenarios(400, 22)
	if len(cases()) != 400 || !reflect.DeepEqual(a, scenarios(400, 11)) {
		t.Fatal("scenario schedule changed")
	}
	seen, configurations := map[int64]bool{}, map[string]bool{}
	for _, scenario := range a {
		seen[scenario.Settings.DealSeed] = true
		configurations[scenario.Opponents] = true
	}
	for _, scenario := range b {
		if seen[scenario.Settings.DealSeed] {
			t.Fatal("split deal overlap")
		}
	}
	if len(configurations) != 4 || len(trials()) != 24 {
		t.Fatal("missing experimental conditions")
	}
}
