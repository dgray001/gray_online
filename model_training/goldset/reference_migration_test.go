package goldset

import (
	"github.com/gin-gonic/gin"
	"os"
	"path/filepath"
	"testing"
)

func TestLegacyNotesAreBackfilledFromStoredReference(t *testing.T) {
	dir := t.TempDir()
	s, err := OpenStore(dir)
	if err != nil {
		t.Fatal(err)
	}
	item := Item{ID: "case", CreatedAt: "captured", Legal: []int{0, 1}, Model4Action: 1, Analysis: gin.H{"raw_bid": 0.3}, State: State{Objective: "win"}}
	if _, err := s.Add([]Item{item}); err != nil {
		t.Fatal(err)
	}
	note := "Original notes\nwith \"quotes\" and probabilities 33%."
	old := []Label{{ItemID: "case", Action: 0, Confidence: 0.8, Notes: note, SavedAt: "saved"}}
	if err := saveJSON(filepath.Join(dir, "labels.json"), old); err != nil {
		t.Fatal(err)
	}
	s, err = OpenStore(dir)
	if err != nil {
		t.Fatal(err)
	}
	label := s.labels[0]
	if label.Notes != note || label.SavedAt != "saved" || label.Action != 0 || label.Confidence != 0.8 || label.Reference == nil || label.Reference.CapturedAt != "captured" || label.Reference.SampledAction != 1 {
		t.Fatal("annotation changed during migration", label)
	}
	if string(label.Reference.Parameters) != "null" {
		t.Fatal("invented unavailable model parameters")
	}
	if _, err := os.Stat(filepath.Join(dir, "labels.before-reference.json")); err != nil {
		t.Fatal("missing original backup", err)
	}
}
