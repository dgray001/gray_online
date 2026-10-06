package combat

import (
	"github.com/dgray001/gray_online/game/games/risq/internal/defs"
	"github.com/dgray001/gray_online/game/games/risq/tests/harness"
	"testing"
)

func TestGameEndClearsDefeatedAndWinnerOrders(t *testing.T) {
	g := defeatBehaviorGame(t, 11)
	owner, winner, last := g.Human(0), g.Human(1), g.Human(2)
	building, worker := defeatBuilding(g, 0, 11), g.Self(owner).Units[0]
	g.Submit(owner, harness.Order(defs.OrderType_BuildingResearch, []uint64{building.InternalID}, 2, false), harness.Order(defs.OrderType_UnitDelete, []uint64{worker.InternalID}, 0, false))
	g.EndTurn()
	if !g.Self(owner).Eliminated || len(g.Self(owner).ActiveOrders) != 1 {
		t.Fatal("fixture did not preserve defeated research order")
	}
	config := defs.TechConfigs[1]
	config.Research_stamina = 30
	defs.TechConfigs[1] = config
	if defeatBuilding(g, 1, 1).CurrentStamina >= 30 {
		t.Fatal("fixture could complete winner research")
	}
	orders := []defs.OrderFromFrontend{harness.Order(defs.OrderType_BuildingResearch, []uint64{defeatBuilding(g, 1, 1).InternalID}, 1, false)}
	for _, unit := range g.Self(winner).Units {
		if unit.UnitID == 13 {
			orders = append(orders, harness.OrderAttackUnit([]uint64{unit.InternalID}, g.Self(last).Units[0].InternalID))
		} else {
			orders = append(orders, harness.OrderMoveZone([]uint64{unit.InternalID}, 2, 0, 0, 0))
		}
	}
	g.Submit(winner, orders...)
	g.Submit(last)
	if !g.Base.GameEnded() || len(g.Self(winner).Units) != 2 || len(g.Self(owner).Buildings) != 1 {
		t.Fatal("game did not end with surviving entities intact")
	}
	for slot := range 3 {
		state := g.Self(g.Human(slot))
		if len(state.ActiveOrders) != 0 {
			t.Fatalf("slot %d retained orders", slot)
		}
		for _, unit := range state.Units {
			if len(unit.ActiveOrders) != 0 {
				t.Fatal("unit retained orders")
			}
		}
		for _, building := range state.Buildings {
			if len(building.ProductionQueue) != 0 {
				t.Fatal("production survived game end")
			}
		}
	}
	if _, pending := g.Self(owner).ResearchedTechs[2]; pending {
		t.Fatal("defeated research survived game end")
	}
	if _, pending := g.Self(winner).ResearchedTechs[1]; pending {
		t.Fatal("winner research survived game end")
	}
	if balance := g.Self(owner).Resources; balance.Stone != 1000 || balance.Gold != 1000 {
		t.Fatalf("defeated research not refunded: %+v", balance)
	}
	if balance := g.Self(winner).Resources; balance.Food != 1000 || balance.Wood != 1000 {
		t.Fatalf("winner research not refunded: %+v", balance)
	}
}
