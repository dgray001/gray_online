package unit

import (
	"strconv"
	"testing"

	. "github.com/dgray001/gray_online/game/games/risq/ai"
)

type alwaysNewRevisionView struct {
	selectTestView
	reads uint64
}

func (v *alwaysNewRevisionView) Revision() uint64 { v.reads++; return v.reads }

func countWhere(kind string, conditions string) string {
	return rule(`{"action":"set_var","name":"n","value":{"count":{"kind":"` + kind + `","where":{"if":` + conditions + `}}}}`)
}

func zonesAcrossFiveSpaces(n int) []ZoneInfo {
	zones := make([]ZoneInfo, n)
	for i := range zones {
		zones[i] = ZoneInfo{Location: ZoneRef{Space: Coordinate{X: i % 5}}, BuildingPlayer: -1}
	}
	return zones
}

func TestSpaceAndZoneJoinsGiveTheSameAnswerAsTestingEveryItem(t *testing.T) {
	view := selectTestView{loopTestView: loopTestView{spaces: lineOfSpaces(make([]float64, 10)...), zones: zonesAcrossFiveSpaces(10)}, turn: 1}
	conditions := []string{"var(item.x) == 2", "(var(item.x) == 2) * (var(item.y) == 0)", "(var(item.x) == 2) * (var(item.y) == 1)"}
	for kind, wants := range map[string][]float64{"spaces": {1, 1, 0}, "zones": {2, 2, 0}} {
		for i, condition := range conditions {
			keyed := decide(t, countWhere(kind, strconv.Quote(condition)), view, "n")["n"]
			scanned := decide(t, countWhere(kind, `[`+strconv.Quote(condition)+`,"random() >= 0"]`), view, "n")["n"]
			if keyed != wants[i] || scanned != keyed {
				t.Errorf("%s where %s: keyed %v, scanned %v, want %v for both", kind, condition, keyed, scanned, wants[i])
			}
		}
	}
}

func TestSpaceAndZoneJoinsSurviveRevisionChangesAndStayWithinTheStepBudget(t *testing.T) {
	const n = 400
	spaces, zones := make([]SpaceInfo, n), make([]ZoneInfo, n)
	for i := range spaces {
		spaces[i] = SpaceInfo{Space: Coordinate{X: i}, Vision: 3, Owner: -1}
		zones[i] = ZoneInfo{Location: ZoneRef{Space: Coordinate{X: i}}, BuildingPlayer: -1}
	}
	for _, kind := range []string{"spaces", "zones"} {
		view := &alwaysNewRevisionView{selectTestView: selectTestView{loopTestView: loopTestView{spaces: spaces, zones: zones}, turn: 1}}
		config := rule(`{"action":"set_var","name":"total","value":0},
			{"action":"for_each","select":{"kind":"` + kind + `"},"as":"node","do":[
			{"action":"set_var","name":"hit","value":{"count":{"kind":"` + kind + `","where":{"if":"var(item.x) == var(node.x)"}}}},
			{"action":"set_var","name":"total","value":"var(total) + var(hit)"}]}`)
		if got := decide(t, config, view, "total")["total"]; got != n {
			t.Errorf("%s: total %v, want %d: a lookup per item must not rescan or rebuild the index", kind, got, n)
		}
	}
}
