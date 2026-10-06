package economy

import (
	"testing"

	"github.com/dgray001/gray_online/game/games/risq/internal/defs"
	"github.com/dgray001/gray_online/game/games/risq/tests/harness"
)

func TestUnaffordableOrdersLeaveNoReservations(t *testing.T) {
	for name, typ := range map[string]defs.OrderType{"building": defs.OrderType_UnitBuild, "unit": defs.OrderType_BuildingCreate, "research": defs.OrderType_BuildingResearch} {
		t.Run(name, func(t *testing.T) {
			g := economyGame(t, `{"food":49,"wood":29}`, `{"x":0,"y":0,"terrain":1,"zones":[{"x":0,"y":0,"building":{"id":1,"player":0},"units":[{"id":1,"player":0,"count":1}]}]},`+remoteVillager, "")
			p := g.Human(0)
			subject, target := g.Self(p).Buildings[0].InternalID, int64(1)
			if typ == defs.OrderType_UnitBuild {
				subject, target = g.Self(p).Units[0].InternalID, harness.BuildKey(2, 0, 0, 1, 0)
			}
			g.Submit(p, harness.Order(typ, []uint64{subject}, target, false))
			g.EndTurn()
			state := g.Self(p)
			wantReason := map[string]string{"building": "cannot afford building", "unit": "cannot afford unit", "research": "cannot afford research"}[name]
			if reasons := state.Refusals(); len(reasons) != 1 || reasons[0] != wantReason {
				t.Errorf("refusals %v, want %q", reasons, wantReason)
			}
			if state.Resources.Food != 49 || state.Resources.Wood != 29 || len(state.Units) != 1 || len(state.Buildings) != 1 || len(state.PlannedFoundations) != 0 || len(state.ActiveOrders) != 0 || len(state.Buildings[0].ProductionQueue) != 0 || len(state.ResearchedTechs) != 0 {
				t.Errorf("unaffordable order changed state: %+v", state)
			}
		})
	}
}
