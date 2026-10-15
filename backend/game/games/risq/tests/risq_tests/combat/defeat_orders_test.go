package combat

import (
	"fmt"
	"testing"

	"github.com/dgray001/gray_online/game/games/risq/internal/defs"
	"github.com/dgray001/gray_online/game/games/risq/tests/harness"
	"github.com/dgray001/gray_online/game/games/risq/tests/harness/fakeboard"
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

func defeatBehaviorGame(t *testing.T, building uint32) *harness.Game {
	t.Helper()
	doc := fmt.Sprintf(`{"board_size":3,"players":3,"starting_bank":{"food":1000,"wood":1000,"stone":1000,"gold":1000},"space_gold_income":0,"spaces":[{"x":0,"y":0,"terrain":1,"zones":[{"x":0,"y":0,"building":{"id":%d,"player":0},"units":[{"id":1,"player":0,"count":1},{"id":13,"player":1,"count":1},{"id":1,"player":2,"count":1}]}]},{"x":-1,"y":0,"terrain":1,"zones":[{"x":0,"y":0,"building":{"id":1,"player":1},"units":[{"id":12,"player":1,"count":1}]}]},{"x":1,"y":0,"terrain":1},{"x":2,"y":0,"terrain":1}]}`, building)
	fakeboard.UseConfig(t, testConfig, nil, map[string]string{"defeat-behavior": doc})
	g := harness.NewGame(t, "custom:defeat-behavior", 1, 3)
	for _, unit := range g.Self(g.Human(1)).Units {
		makePassive(g, 1, unit)
	}
	return g
}

func defeatBuilding(g *harness.Game, slot int, id uint32) harness.Building {
	g.T.Helper()
	for _, building := range g.Self(g.Human(slot)).Buildings {
		if building.BuildingID == id {
			return building
		}
	}
	g.T.Fatalf("missing building %d for slot %d", id, slot)
	return harness.Building{}
}

func TestDefeatWithRemainingBuildings(t *testing.T) {
	for _, id := range []uint32{1, 2, 3, 11, 21, 22} {
		t.Run(fmt.Sprint(id), func(t *testing.T) {
			doc := fmt.Sprintf(`{"board_size":1,"players":2,"spaces":[{"x":0,"y":0,"terrain":1,"zones":[{"x":0,"y":0,"building":{"id":%d,"player":0},"units":[{"id":1,"player":1,"count":1}]}]}]}`, id)
			fakeboard.UseConfig(t, testConfig, nil, map[string]string{"defeat": doc})
			g := harness.NewGame(t, "custom:defeat", 1, 2)
			p := g.Self(g.Human(0))
			defeated := id != 1 && id != 22
			if p.Eliminated != defeated || g.Base.GameEnded() != defeated || len(p.Buildings) != 1 {
				t.Fatalf("building %d: defeated=%v ended=%v buildings=%d", id, p.Eliminated, g.Base.GameEnded(), len(p.Buildings))
			}
		})
	}
}

func TestUnfinishedProducerDoesNotPreventDefeat(t *testing.T) {
	g := defeatBehaviorGame(t, 2)
	owner := g.Human(0)
	worker := g.Self(owner).Units[0]
	g.Submit(owner, harness.OrderBuild([]uint64{worker.InternalID}, 22, 0, 0, -1, 0))
	g.EndTurn()
	if !defeatBuilding(g, 0, 22).UnderConstruction {
		t.Fatal("fixture did not leave unfinished barracks")
	}
	g.Submit(owner, harness.Order(defs.OrderType_UnitDelete, []uint64{worker.InternalID}, 0, true))
	g.EndTurn()
	if state := g.Self(owner); !state.Eliminated || g.Base.GameEnded() {
		t.Fatalf("unfinished producer should not prevent defeat: %+v", state)
	}
}
