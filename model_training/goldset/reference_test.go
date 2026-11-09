package goldset

import (
	"encoding/json"
	"github.com/gin-gonic/gin"
	"testing"
)

func TestReferenceSnapshotDoesNotChangeWithModelOutput(t *testing.T) {
	item := Item{ID: "case", CreatedAt: "original", Model4Action: 1, Analysis: gin.H{"raw_bid": 0.3}, State: State{Objective: "win"}}
	a, err := freezeReference(item, "item-snapshot")
	if err != nil {
		t.Fatal(err)
	}
	same, _ := freezeReference(item, "stored-item-backfill")
	if a.SHA256 != same.SHA256 {
		t.Fatal("origin changed content fingerprint")
	}
	item.Analysis["raw_bid"] = 0.9
	item.Model4Action = 0
	b, _ := freezeReference(item, "item-snapshot")
	var original map[string]float64
	if err := json.Unmarshal(a.Analysis, &original); err != nil {
		t.Fatal(err)
	}
	if original["raw_bid"] != 0.3 || a.SampledAction != 1 || a.SHA256 == b.SHA256 {
		t.Fatal("reference changed with live output")
	}
}
