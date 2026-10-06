package combat

import (
	"github.com/dgray001/gray_online/game"
	"github.com/dgray001/gray_online/game/games/risq/internal/defs"
	"github.com/dgray001/gray_online/game/games/risq/tests/harness"
	"reflect"
	"testing"
)

func TestDefeatedBuildingsStayFrozenAndAttackable(t *testing.T) {
	g := defeatBehaviorGame(t, 11)
	owner := g.Human(0)
	building := defeatBuilding(g, 0, 11)
	worker := g.Self(owner).Units[0]
	g.Submit(owner, harness.Order(defs.OrderType_BuildingResearch, []uint64{building.InternalID}, 2, false), harness.Order(defs.OrderType_UnitDelete, []uint64{worker.InternalID}, 0, false))
	g.EndTurn()
	before := g.Self(owner)
	if !before.Eliminated || len(before.Buildings[0].ProductionQueue) != 1 || len(before.ActiveOrders) != 1 {
		t.Fatalf("defeat fixture=%+v", before)
	}
	for _, kind := range []string{"unsubmit-orders", "set-unit-behavior", "set-building-behavior", "set-gather-point"} {
		g.Risq.PlayerAction(game.PlayerAction{Client_id: owner + 1, Kind: kind})
		if failures := g.Failed(owner); len(failures) != 1 || failures[0] != "Eliminated players cannot act" {
			t.Fatalf("%s failures=%v", kind, failures)
		}
	}
	turn := g.State(owner).TurnNumber
	g.Submit(g.Human(1))
	g.Submit(g.Human(2))
	after := g.Self(owner)
	if g.State(owner).TurnNumber != turn+1 || !reflect.DeepEqual(before.Buildings, after.Buildings) || !reflect.DeepEqual(before.ActiveOrders, after.ActiveOrders) || *before.Resources != *after.Resources {
		t.Fatalf("defeated state changed: before=%+v after=%+v", before, after)
	}
	var attacker uint64
	for _, unit := range g.Self(g.Human(1)).Units {
		if unit.UnitID == 13 {
			attacker = unit.InternalID
		}
	}
	g.Submit(g.Human(1), harness.Order(defs.OrderType_UnitAttackBuilding, []uint64{attacker}, int64(building.InternalID), false))
	g.Submit(g.Human(2))
	if defeatBuilding(g, 0, 11).CombatStats.Health >= after.Buildings[0].CombatStats.Health {
		t.Fatal("defeated building became immune to damage")
	}
}
