package goldset

import (
	"slices"
	"testing"
)

func TestKnownChecksWinAcrossIndependentLegalCompletions(t *testing.T) {
	for _, spec := range foundationChecks() {
		item, err := foundationItem(spec, 123, 1)
		if err != nil {
			t.Fatal(spec.Name, err)
		}
		key, err := foundationKey(spec, item)
		if err != nil {
			t.Fatal(err)
		}
		for seed := int64(1); seed <= 12; seed++ {
			for _, action := range item.Legal {
				f, err := prepareFoundation(spec, seed)
				if err != nil {
					t.Fatal(spec.Name, err)
				}
				if err := f.ApplyLearningAction(action); err != nil {
					t.Fatal(err)
				}
				for !f.ExecuteAiTurn(false) {
				}
				_, scores := f.GetGameResults()
				wins := scores[spec.Seat] == slices.Max(scores)
				if wins != slices.Contains(key.Acceptable, action) {
					t.Fatalf("check proof failed for %s seed %d", spec.Name, seed)
				}
			}
		}
	}
}
