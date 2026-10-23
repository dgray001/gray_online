package economy

import (
	"testing"

	"github.com/dgray001/gray_online/game/games/risq/internal/defs"
	"github.com/dgray001/gray_online/game/games/risq/tests/harness"
)

// Refused orders leave the state untouched, so the three kinds run back to back in one game
func TestUnaffordableOrdersLeaveNoReservations(t *testing.T) {
	g := economyGame(t, `{"food":49,"wood":29}`, `{"x":0,"y":0,"terrain":1,"zones":[{"x":0,"y":0,"building":{"id":201,"player":0},"units":[{"id":11,"player":0,"count":1}]}]},`+remoteWorker, "")
	p := g.Human(0)
	for _, c := range []struct {
		name   string
		typ    defs.OrderType
		reason string
	}{
		{"building", defs.OrderType_UnitBuild, "cannot afford building"},
		{"unit", defs.OrderType_BuildingCreate, "cannot afford unit"},
		{"research", defs.OrderType_BuildingResearch, "cannot afford research"},
	} {
		subject, target := g.Self(p).Buildings[0].InternalID, int64(11)
		if c.typ == defs.OrderType_UnitBuild {
			subject, target = g.Self(p).Units[0].InternalID, harness.BuildKey(202, 0, 0, 1, 0)
		} else if c.typ == defs.OrderType_BuildingResearch {
			target = 301
		}
		g.Submit(p, harness.Order(c.typ, []uint64{subject}, target, false))
		g.EndTurn()
		state := g.Self(p)
		if reasons := state.Refusals(); len(reasons) != 1 || reasons[0] != c.reason {
			t.Errorf("%s: refusals %v, want %q", c.name, reasons, c.reason)
		}
		if state.Resources.Food != 49 || state.Resources.Wood != 29 || len(state.Units) != 1 || len(state.Buildings) != 1 || len(state.PlannedFoundations) != 0 || len(state.ActiveOrders) != 0 || len(state.Buildings[0].ProductionQueue) != 0 || len(state.ResearchedTechs) != 0 {
			t.Errorf("%s: unaffordable order changed state: %+v", c.name, state)
		}
	}
}

func TestOrderBatchesCannotOverspend(t *testing.T) {
	for name, typ := range map[string]defs.OrderType{"building": defs.OrderType_UnitBuild, "unit": defs.OrderType_BuildingCreate, "research": defs.OrderType_BuildingResearch} {
		t.Run(name, func(t *testing.T) {
			bank, building := `{"food":80,"wood":50}`, "201"
			if name == "research" {
				bank, building = `{"wood":150,"gold":150}`, "209"
			}
			g := economyGame(t, bank, `{"x":0,"y":0,"terrain":1,"zones":[{"x":0,"y":0,"building":{"id":`+building+`,"player":0},"units":[{"id":11,"player":0,"count":1}]}]},`+remoteWorker, "")
			p := g.Human(0)
			subject, first, second := g.Self(p).Buildings[0].InternalID, int64(11), int64(11)
			if name == "building" {
				subject = g.Self(p).Units[0].InternalID
				first, second = harness.BuildKey(202, 0, 0, 1, 0), harness.BuildKey(202, 0, 0, -1, 0)
			} else if name == "research" {
				first, second = 303, 305
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
				if state.Resources.Wood != 50 || state.Resources.Gold != 70 || len(state.Buildings[0].ProductionQueue) != 1 || len(state.ResearchedTechs) != 1 {
					t.Errorf("research batch overspent or reserved unpaid work: %+v", state)
				}
			}
		})
	}
}
