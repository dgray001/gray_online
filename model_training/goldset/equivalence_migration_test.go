package goldset

import (
	"path/filepath"
	"slices"
	"testing"
)

func TestEquivalenceMigrationOnlyChangesLatestAndDoesNotReapply(t *testing.T) {
	dir := t.TempDir()
	s, _ := OpenStore(dir)
	s.Add([]Item{{ID: "equal", Legal: []int{34, 36}}, {ID: "uncertain", Legal: []int{0, 1}}})
	original := []Label{{ItemID: "equal", Action: 34, Confidence: 1, Notes: "older choice", SavedAt: "old"}, {ItemID: "equal", Action: 36, Confidence: 0.5, Notes: "both equal", SavedAt: "saved"}, {ItemID: "uncertain", Action: 1, Confidence: 0.5, Notes: "50/50 chance", SavedAt: "saved"}}
	saveJSON(filepath.Join(dir, "labels.json"), original)
	saveJSON(filepath.Join(dir, "equivalences.json"), map[string][]int{"equal": {34, 36}})
	s, err := OpenStore(dir)
	if err != nil {
		t.Fatal(err)
	}
	if !slices.Equal(s.labels[0].Acceptable, []int{34}) || !slices.Equal(s.labels[1].Acceptable, []int{34, 36}) || !slices.Equal(s.labels[2].Acceptable, []int{1}) {
		t.Fatal("incorrect migration", s.labels)
	}
	for i, label := range s.labels {
		if label.Action != original[i].Action || label.Notes != original[i].Notes || label.SavedAt != original[i].SavedAt || label.Confidence != original[i].Confidence {
			t.Fatal("original label changed")
		}
	}
	if err := s.SaveLabel("equal", Label{Action: 36, Acceptable: []int{36}, Confidence: 1, Notes: "revised"}); err != nil {
		t.Fatal(err)
	}
	s, err = OpenStore(dir)
	if err != nil {
		t.Fatal(err)
	}
	if !slices.Equal(s.latest["equal"].Acceptable, []int{36}) {
		t.Fatal("old correction overwrote later human revision")
	}
}
