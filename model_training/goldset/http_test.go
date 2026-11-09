package goldset

import (
	"encoding/json"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestLabelAPIAndExport(t *testing.T) {
	store, err := OpenStore(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	store.Add([]Item{{ID: "case", SchemaVersion: 2, Legal: []int{0, 1}, Model4Action: 1, State: State{Kind: "bet", Objective: "win"}}})
	handler := Handler(store, t.TempDir(), t.TempDir())
	bad := httptest.NewRecorder()
	handler.ServeHTTP(bad, httptest.NewRequest("POST", "/api/items/case/label", strings.NewReader(`{"action":78,"confidence":1}`)))
	if bad.Code != 400 {
		t.Fatal("accepted illegal move")
	}
	saved := httptest.NewRecorder()
	handler.ServeHTTP(saved, httptest.NewRequest("POST", "/api/items/case/label", strings.NewReader(`{"action":0,"confidence":0.8,"notes":"human reasoning"}`)))
	if saved.Code != 200 {
		t.Fatal(saved.Body.String())
	}
	export := httptest.NewRecorder()
	handler.ServeHTTP(export, httptest.NewRequest("GET", "/api/export", nil))
	var data Goldset
	if err := json.Unmarshal(export.Body.Bytes(), &data); err != nil {
		t.Fatal(err)
	}
	if len(data.Examples) != 1 || data.Examples[0].Target != 0 || data.Examples[0].Notes != "human reasoning" {
		t.Fatal(data)
	}
}
