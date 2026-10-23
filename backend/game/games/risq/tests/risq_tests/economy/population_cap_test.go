package economy

import (
	"testing"

	"github.com/dgray001/gray_online/game/games/risq/internal/defs"
	"github.com/dgray001/gray_online/game/games/risq/tests/harness"
)

func TestPopulationCapPausesAndResumesProduction(t *testing.T) {
	g := productionGame(t, 5, 1, "")
	p := g.Human(0)
	g.Submit(p, harness.OrderProduce([]uint64{centers(g, p)[0].InternalID}, 11))
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
	g := economyGameTweaked(t, `{}`, `{"x":0,"y":0,"terrain":1,"zones":[{"x":0,"y":0,"building":{"id":208,"player":0},"units":[{"id":11,"player":0,"count":1}]}]},{"x":1,"y":0,"terrain":1,"zones":[{"x":0,"y":0,"building":{"id":208,"player":0}}]},{"x":2,"y":0,"terrain":1,"zones":[{"x":0,"y":0,"building":{"id":208,"player":0}}]},`+remoteWorker, "", func() {
		fortress := defs.BuildingConfigs[208]
		fortress.Population_support = 50
		defs.BuildingConfigs[208] = fortress
	})
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
	g.Submit(p, harness.OrderProduce([]uint64{centers(g, p)[0].InternalID}, 11), harness.Order(defs.OrderType_BuyMercenary, nil, harness.BuildKey(21, 0, 0, 0, 0), false))
	g.EndTurn()
	state := g.Self(p)
	if len(state.Units) != 5 || state.Resources.Food != 250 || state.Resources.Gold != 500-gruntMercenaryGold || len(centers(g, p)[0].ProductionQueue) != 1 || centers(g, p)[0].ProductionQueue[0].StaminaRemaining != 10 {
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
	g.Submit(p, harness.OrderProduce([]uint64{buildings[0].InternalID}, 11), harness.OrderProduce([]uint64{buildings[1].InternalID}, 11))
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
				if item.ItemID != 11 || item.StaminaRemaining <= 0 {
					t.Errorf("pending production %+v, want unfinished worker", item)
				}
			}
		}
	}
}

func TestShelterSupportsPopulationOnlyWhileCompletedAndAlive(t *testing.T) {
	g := buildGame(t, 1)
	p := g.Human(0)
	g.Submit(p, shelterOrders(g, p, 0)...)
	g.EndTurn()
	if g.Self(p).PopulationLimit != 0 || !centerShelter(g, p).UnderConstruction {
		t.Fatal("unfinished shelter provided population support")
	}
	g.EndTurn()
	if g.Self(p).PopulationLimit != 5 || centerShelter(g, p).UnderConstruction {
		t.Fatal("completed shelter did not provide population support")
	}
	g.Submit(p, harness.Order(defs.OrderType_BuildingDelete, []uint64{centerShelter(g, p).InternalID}, 0, false))
	g.EndTurn()
	if g.Self(p).PopulationLimit != 0 || len(g.Self(p).Buildings) != 0 {
		t.Error("deleted shelter retained population support")
	}
}
