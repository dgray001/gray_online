package economy

import (
	"testing"

	"github.com/dgray001/gray_online/game/games/risq/internal/defs"
	"github.com/dgray001/gray_online/game/games/risq/tests/harness"
)

func TestOrderBatchesCannotOverspend(t *testing.T) {
	for name, typ := range map[string]defs.OrderType{"building": defs.OrderType_UnitBuild, "unit": defs.OrderType_BuildingCreate, "research": defs.OrderType_BuildingResearch} {
		t.Run(name, func(t *testing.T) {
			bank, building := `{"food":80,"wood":50}`, "1"
			if name == "research" {
				bank, building = `{"stone":150,"gold":150}`, "11"
			}
			g := economyGame(t, bank, `{"x":0,"y":0,"terrain":1,"zones":[{"x":0,"y":0,"building":{"id":`+building+`,"player":0},"units":[{"id":1,"player":0,"count":1}]}]},`+remoteVillager, "")
			p := g.Human(0)
			subject, first, second := g.Self(p).Buildings[0].InternalID, int64(1), int64(1)
			if name == "building" {
				subject = g.Self(p).Units[0].InternalID
				first, second = harness.BuildKey(2, 0, 0, 1, 0), harness.BuildKey(2, 0, 0, -1, 0)
			} else if name == "research" {
				first, second = 2, 3
			}
			g.Submit(p, harness.Order(typ, []uint64{subject}, first, false), harness.Order(typ, []uint64{subject}, second, false))
			g.EndTurn()
			state := g.Self(p)
			wantReason := map[string]string{"building": "cannot afford building", "unit": "cannot afford unit", "research": "cannot afford research"}[name]
			if reasons := state.Refusals(); len(reasons) != 1 || reasons[0] != wantReason {
				t.Fatalf("refusals %v, want %q", reasons, wantReason)
			}
			switch name {
			case "building":
				if state.Resources.Wood != 20 || len(state.Buildings)-1+len(state.PlannedFoundations) != 1 || len(state.ActiveOrders) != 1 {
					t.Errorf("building batch overspent or reserved unpaid work: %+v", state)
				}
			case "unit":
				if state.Resources.Food != 30 || len(state.Units) != 2 || len(state.Buildings[0].ProductionQueue) != 0 {
					t.Errorf("unit batch overspent or queued unpaid work: %+v", state)
				}
			case "research":
				if state.Resources.Stone != 50 || state.Resources.Gold != 50 || len(state.Buildings[0].ProductionQueue) != 1 || len(state.ResearchedTechs) != 1 {
					t.Errorf("research batch overspent or reserved unpaid work: %+v", state)
				}
			}
		})
	}
}
