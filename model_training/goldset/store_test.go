package goldset

import (
	"encoding/json"
	"fiddlesticks.live/neural"
	"github.com/gin-gonic/gin"
	"strings"
	"testing"
)

func TestLabelsPersistAndExportHumanChoicesOnly(t *testing.T) {
	dir := t.TempDir()
	store, err := OpenStore(dir)
	if err != nil {
		t.Fatal(err)
	}
	item := Item{ID: "case", SchemaVersion: 2, Features: make([]float64, neural.Inputs), Legal: []int{0, 1}, Model4Action: 1, Analysis: gin.H{"raw_bid": 0.3}, State: State{Kind: "bet", Objective: "win"}}
	if _, err := store.Add([]Item{item}); err != nil {
		t.Fatal(err)
	}
	if err := store.SaveLabel("case", Label{Action: 0, Confidence: 0.8, Notes: "prefer zero"}); err != nil {
		t.Fatal(err)
	}
	reopened, err := OpenStore(dir)
	if err != nil {
		t.Fatal(err)
	}
	result := reopened.Export()
	if len(result.Examples) != 1 || result.Examples[0].Target != 0 || result.Objective != "win" || result.Inputs != 740 {
		t.Fatal(result)
	}
	data, _ := json.Marshal(result)
	if !strings.Contains(string(data), "model4_reference") || result.Examples[0].Reference == nil || result.Examples[0].Reference.SampledAction != 1 {
		t.Fatal("reference context missing or confused with human target")
	}
	if err := reopened.SaveLabel("case", Label{Action: 1, Confidence: 1}); err != nil {
		t.Fatal(err)
	}
	latest, _ := OpenStore(dir)
	if len(latest.labels) != 2 || latest.Export().Examples[0].Target != 1 {
		t.Fatal("revision lost")
	}
	if err := latest.SaveLabel("case", Label{Action: 78, Confidence: 1}); err == nil {
		t.Fatal("illegal label accepted")
	}
}
