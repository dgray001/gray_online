package economy

import (
	"testing"

	"github.com/dgray001/gray_online/game/games/risq/internal/defs"
	"github.com/dgray001/gray_online/game/games/risq/tests/harness"
)

func TestPopulationCapPausesAndResumesProduction(t *testing.T) {
	g := productionGame(t, 5, 1, "")
	p := g.Human(0)
	g.Submit(p, harness.OrderProduce([]uint64{centers(g, p)[0].InternalID}, 1))
	g.EndTurn()
	queue := centers(g, p)[0].ProductionQueue
	if state := g.Self(p); len(state.Units) != 5 || state.PopulationLimit != 5 || state.Resources.Food != 250 || len(queue) != 1 || queue[0].StaminaRemaining != 10 {
		t.Fatalf("capped production: units %d, cap %d, food %v, queue %v", len(state.Units), state.PopulationLimit, state.Resources.Food, queue)
	}
	deleted := g.Self(p).Units[0].InternalID
	g.Submit(p, harness.Order(defs.OrderType_UnitDelete, []uint64{deleted}, 0, false))
	for turn := 0; turn < 2; turn++ {
		g.EndTurn()
		if state := g.Self(p); len(state.Units) > state.PopulationLimit {
			t.Fatal("production exceeded the population cap")
		}
	}
	state := g.Self(p)
	if len(state.Units) != 5 || state.Resources.Food != 250 || len(centers(g, p)[0].ProductionQueue) != 0 || g.State(p).Unit(deleted) != nil {
		t.Errorf("resumed production: units %d, food %v, queue %v; want replacement at cap with no new charge", len(state.Units), state.Resources.Food, centers(g, p)[0].ProductionQueue)
	}
}

func TestPopulationSupportIsClampedToGameMaximum(t *testing.T) {
	g := economyGame(t, `{}`, `{"x":0,"y":0,"terrain":1,"zones":[{"x":0,"y":0,"building":{"id":23,"player":0},"units":[{"id":1,"player":0,"count":1}]}]},{"x":1,"y":0,"terrain":1,"zones":[{"x":0,"y":0,"building":{"id":23,"player":0}}]},{"x":2,"y":0,"terrain":1,"zones":[{"x":0,"y":0,"building":{"id":23,"player":0}}]},`+remoteVillager, "")
	p := g.Human(0)
	if g.Self(p).PopulationLimit != 100 {
		t.Fatal("150 support did not clamp to the 100-unit population limit")
	}
	for _, want := range []int{100, 50} {
		g.Submit(p, harness.Order(defs.OrderType_BuildingDelete, []uint64{g.Self(p).Buildings[0].InternalID}, 0, false))
		g.EndTurn()
		if g.Self(p).PopulationLimit != want {
			t.Errorf("remaining support produced limit %d, want %d", g.Self(p).PopulationLimit, want)
		}
	}
}

func TestMercenaryReservationBlocksProductionAtCap(t *testing.T) {
	g := productionGame(t, 4, 1, mercenaryRules)
	p := g.Human(0)
	g.Submit(p, harness.OrderProduce([]uint64{centers(g, p)[0].InternalID}, 1), harness.Order(defs.OrderType_BuyMercenary, nil, harness.BuildKey(11, 0, 0, 0, 0), false))
	g.EndTurn()
	state := g.Self(p)
	if len(state.Units) != 5 || state.Resources.Food != 250 || state.Resources.Gold != 409 || len(centers(g, p)[0].ProductionQueue) != 1 || centers(g, p)[0].ProductionQueue[0].StaminaRemaining != 10 {
		t.Fatalf("production ignored reserved slot: %+v", state)
	}
}

func TestSimultaneousProductionRespectsLastPopulationSlot(t *testing.T) {
	g := productionGame(t, 9, 2, "")
	p := g.Human(0)
	buildings := centers(g, p)
	winner := buildings[0].InternalID
	if buildings[1].InternalID < winner {
		winner = buildings[1].InternalID
	}
	g.Submit(p, harness.OrderProduce([]uint64{buildings[0].InternalID}, 1), harness.OrderProduce([]uint64{buildings[1].InternalID}, 1))
	for turn := 0; turn < 2; turn++ {
		g.EndTurn()
		state := g.Self(p)
		queued := len(centers(g, p)[0].ProductionQueue) + len(centers(g, p)[1].ProductionQueue)
		if len(state.Units) != 10 || state.PopulationLimit != 10 || queued != 1 || state.Resources.Food != 200 {
			t.Errorf("turn %d: units %d, cap %d, queued %d, food %v; want 10, 10, 1, 200", turn, len(state.Units), state.PopulationLimit, queued, state.Resources.Food)
		}
		for _, b := range centers(g, p) {
			if (len(b.ProductionQueue) == 0) != (b.InternalID == winner) {
				t.Errorf("building %d won=%v, want winner %d", b.InternalID, len(b.ProductionQueue) == 0, winner)
			}
			for _, item := range b.ProductionQueue {
				if item.ItemID != 1 || item.StaminaRemaining <= 0 {
					t.Errorf("pending production %+v, want unfinished villager", item)
				}
			}
		}
	}
}

func TestHousingSupportsPopulationOnlyWhileCompletedAndAlive(t *testing.T) {
	g := buildGame(t, 1)
	p := g.Human(0)
	g.Submit(p, housingOrders(g, p, 0)...)
	g.EndTurn()
	if g.Self(p).PopulationLimit != 0 || !centerHousing(g, p).UnderConstruction {
		t.Fatal("unfinished housing provided population support")
	}
	g.EndTurn()
	if g.Self(p).PopulationLimit != 5 || centerHousing(g, p).UnderConstruction {
		t.Fatal("completed housing did not provide population support")
	}
	g.Submit(p, harness.Order(defs.OrderType_BuildingDelete, []uint64{centerHousing(g, p).InternalID}, 0, false))
	g.EndTurn()
	if g.Self(p).PopulationLimit != 0 || len(g.Self(p).Buildings) != 0 {
		t.Error("deleted housing retained population support")
	}
}
