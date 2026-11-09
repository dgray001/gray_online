package goldset

import (
	"slices"
	"testing"
)

func TestGenerationCapturesLegalPublicDecisions(t *testing.T) {
	q := GenerateRequest{Count: 6, Players: 4, MinHand: 1, MaxHand: 2, Kind: "both", RoundPoints: 10, TrickPoints: 1, Seed: 42}
	items, err := Generate(q)
	if err != nil {
		t.Fatal(err)
	}
	bids, cards := 0, 0
	for _, item := range items {
		if len(item.Model4Parameters) == 0 {
			t.Fatal("model parameters missing from new capture")
		}
		if len(item.Features) != 740 || item.SchemaVersion != 2 || item.State.Objective != "win" || !slices.Contains(item.Legal, item.Model4Action) {
			t.Fatal("bad snapshot or reference", item.ID)
		}
		if item.State.Kind == "bet" {
			bids++
		} else {
			cards++
		}
		if item.State.Progress.Total != 23 || item.State.MaxHand != 12 {
			t.Fatal("game progress missing", item.State.Progress)
		}
	}
	if bids != 3 || cards != 3 {
		t.Fatal("unbalanced queue", bids, cards)
	}
	again, err := Generate(q)
	if err != nil {
		t.Fatal(err)
	}
	for i := range items {
		if items[i].ID != again[i].ID || items[i].Model4Action != again[i].Model4Action {
			t.Fatal("seed did not reproduce item")
		}
	}
}
