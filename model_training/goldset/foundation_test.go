package goldset

import (
	"encoding/json"
	"path/filepath"
	"slices"
	"strings"
	"testing"
)

func TestFoundationBatchHasThirtyUnlabelledCasesAndPrivateChecks(t *testing.T) {
	s, err := OpenStore(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	added, err := AddFoundationBatch(s)
	if err != nil {
		t.Fatal(err)
	}
	if added != 30 {
		t.Fatal(added)
	}
	bids, cards := 0, 0
	for _, q := range s.Queue() {
		entry, _ := s.Get(q.ID)
		if q.Labelled || len(entry.Features) != 740 || !slices.Contains(entry.Legal, entry.Model4Action) {
			t.Fatal("invalid item", q.ID)
		}
		if q.Kind == "bet" {
			bids++
		} else {
			cards++
		}
	}
	if bids != 15 || cards != 15 || len(s.Export().Examples) != 0 {
		t.Fatal("incorrect phase mix or automatic gold labels")
	}
	var keys []FoundationCheck
	if err := loadJSON(filepath.Join(s.Directory, "batches", "foundation-002-checks.json"), &keys); err != nil {
		t.Fatal(err)
	}
	if len(keys) != 8 {
		t.Fatal("wrong check count")
	}
	for _, key := range keys {
		entry, _ := s.Get(key.ItemID)
		data, _ := json.Marshal(entry)
		if strings.Contains(string(data), key.Proof) || strings.Contains(string(data), "acceptable_actions") {
			t.Fatal("answer key exposed before annotation")
		}
	}
	first := s.Queue()[0]
	entry, _ := s.Get(first.ID)
	if err := s.SaveLabel(first.ID, Label{Action: entry.Legal[0], Confidence: 1, Notes: "human label"}); err != nil {
		t.Fatal(err)
	}
	if added, err := AddFoundationBatch(s); err != nil || added != 0 || len(s.Queue()) != 30 || len(s.Export().Examples) != 1 {
		t.Fatal("batch is not idempotent", added, err)
	}
}
