package goldset

import (
	"slices"
	"testing"
)

func TestAcceptableSetsValidateAndExport(t *testing.T) {
	s, err := OpenStore(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	s.Add([]Item{{ID: "case", Legal: []int{34, 36, 40}, State: State{Kind: "play-card", Objective: "win"}}})
	if err := s.SaveLabel("case", Label{Action: 36, Acceptable: []int{36, 34, 36}, Confidence: 0.5}); err != nil {
		t.Fatal(err)
	}
	data := s.Export()
	if data.LabelVersion != 2 || data.Examples[0].Target != 36 || !slices.Equal(data.Examples[0].Acceptable, []int{34, 36}) {
		t.Fatal(data)
	}
	if err := s.SaveLabel("case", Label{Action: 36, Acceptable: []int{34}, Confidence: 1}); err == nil {
		t.Fatal("primary excluded from acceptable set")
	}
	if err := s.SaveLabel("case", Label{Action: 36, Acceptable: []int{36, 78}, Confidence: 1}); err == nil {
		t.Fatal("illegal equivalent accepted")
	}
}
