package combat

import (
	"fmt"
	"github.com/dgray001/gray_online/game/games/risq/internal/defs"
	"github.com/dgray001/gray_online/game/games/risq/tests/harness"
	"github.com/dgray001/gray_online/game/games/risq/tests/harness/fakeboard"
	"github.com/gin-gonic/gin"
	"testing"
)

func TestStancesCreateAutoAttackBuilding(t *testing.T) {
	for _, stance := range []defs.UnitStance{defs.UnitStance_AGGRESSIVE, defs.UnitStance_DEFENSIVE, defs.UnitStance_STAND_GROUND} {
		t.Run(fmt.Sprint(stance), func(t *testing.T) {
			g := priorityDistanceGame(t)
			owner := g.Human(0)
			unit := g.Self(owner).Units[0]
			g.Action(owner, "set-unit-behavior", gin.H{"internal_ids": []uint64{unit.InternalID}, "stance": uint8(stance), "attack_back": false, "target_priority": []uint8{uint8(defs.TargetCategory_BUILDING)}})
			g.EndTurn()
			orders := g.Self(owner).Units[0].ActiveOrders
			if len(orders) != 1 || defs.OrderType(orders[0].OrderType) != defs.OrderType_UnitAutoAttackBuilding {
				t.Fatalf("stance must create an auto-attack-building order, got %+v", orders)
			}
		})
	}
}

func TestTargetPriority(t *testing.T) {
	// p0 has unit 13. p1 has 10 unit 1 (Villager, Economic) and unit 11 (Blunt Infantry, Infantry).
	doc := `{"board_size":1,"players":2,"spaces":[{"x":0,"y":0,"terrain":1,"zones":[{"x":0,"y":0,"units":[{"id":13,"player":0,"count":1},{"id":1,"player":1,"count":10},{"id":11,"player":1,"count":1}]}]}]}`
	fakeboard.UseConfig(t, testConfig, nil, map[string]string{"combat_priority": doc})
	g := harness.NewGame(t, "custom:combat_priority", 1, 2)
	p0, p1 := g.Human(0), g.Human(1)
	u0 := g.Self(p0).Units[0]

	// Set target priority to prefer ECONOMIC over INFANTRY
	g.Action(p0, "set-unit-behavior", gin.H{
		"internal_ids":    []uint64{u0.InternalID},
		"target_priority": []uint8{uint8(defs.TargetCategory_ECONOMIC), uint8(defs.TargetCategory_MILITARY)},
	})

	// Order attack on space
	g.Submit(p0, harness.Order(defs.OrderType_UnitAttackSpace, []uint64{u0.InternalID}, harness.SpaceKey(0, 0), false))
	g.Submit(p1)

	var infantry *harness.Unit
	survivors := g.Self(p1).Units
	for i := range survivors {
		if survivors[i].UnitID == 11 {
			infantry = &survivors[i]
		}
	}
	if infantry == nil {
		t.Errorf("infantry should have survived, but it was killed")
	} else if infantry.CombatStats.Health < 11 {
		t.Errorf("infantry incorrectly took damage, health %v", infantry.CombatStats.Health)
	}
}

func TestInterruptCurrentTargetPriorityGate(t *testing.T) {
	// Active order is NOT interrupted by a category not in target_priority
	t.Run("unprioritized target does not interrupt active order", func(t *testing.T) {
		g := priorityDistanceGame(t)
		owner, enemy := g.Human(0), g.Human(1)
		attacker := g.Self(owner).Units[0]

		for _, unit := range g.Self(enemy).Units {
			g.Action(enemy, "set-unit-behavior", gin.H{"internal_ids": []uint64{unit.InternalID}, "stance": uint8(defs.UnitStance_PASSIVE), "attack_back": false})
		}

		// Attacker has interrupt_current = true, but empty target_priority.
		// None of the categories (MILITARY, ECONOMIC, BUILDING) are in priority.
		g.Action(owner, "set-unit-behavior", gin.H{
			"internal_ids":      []uint64{attacker.InternalID},
			"stance":            uint8(defs.UnitStance_AGGRESSIVE),
			"attack_back":       false,
			"interrupt_current": true,
			"target_priority":   []uint8{},
		})

		g.Submit(owner, harness.OrderMoveZone([]uint64{attacker.InternalID}, 0, 0, -1, 0))
		g.Submit(enemy)

		orders := g.Self(owner).Units[0].ActiveOrders
		for _, o := range orders {
			if defs.OrderType(o.OrderType).IsAttackOrder() {
				t.Fatalf("expected attacker NOT to interrupt move order for unprioritized target, but got order %+v", o)
			}
		}
	})

	// Active order IS interrupted when target category IS in target_priority
	t.Run("prioritized target does interrupt active order", func(t *testing.T) {
		g := priorityDistanceGame(t)
		owner, enemy := g.Human(0), g.Human(1)
		attacker := g.Self(owner).Units[0]

		for _, unit := range g.Self(enemy).Units {
			g.Action(enemy, "set-unit-behavior", gin.H{"internal_ids": []uint64{unit.InternalID}, "stance": uint8(defs.UnitStance_PASSIVE), "attack_back": false})
		}

		g.Action(owner, "set-unit-behavior", gin.H{
			"internal_ids":      []uint64{attacker.InternalID},
			"stance":            uint8(defs.UnitStance_AGGRESSIVE),
			"attack_back":       false,
			"interrupt_current": true,
			"target_priority":   []uint8{uint8(defs.TargetCategory_BUILDING)},
		})

		g.Submit(owner, harness.OrderMoveZone([]uint64{attacker.InternalID}, 0, 0, -1, 0))
		g.Submit(enemy)

		u := g.Self(owner).Units[0]
		hasAttackOrder := false
		for _, o := range u.ActiveOrders {
			if defs.OrderType(o.OrderType) == defs.OrderType_UnitAutoAttackBuilding {
				hasAttackOrder = true
				break
			}
		}
		b := g.Self(enemy).Buildings[0]
		if !hasAttackOrder && b.CombatStats.Health >= float64(b.CombatStats.MaxHealth) {
			t.Fatalf("expected attacker to interrupt move order and attack building, building health=%v, orders=%+v", b.CombatStats.Health, u.ActiveOrders)
		}
	})

	// Idle unit (len(active_orders) == 0) CAN attack unprioritized target via proximity fallback
	t.Run("idle unit attacks unprioritized target via fallback", func(t *testing.T) {
		g := priorityDistanceGame(t)
		owner, enemy := g.Human(0), g.Human(1)
		attacker := g.Self(owner).Units[0]

		for _, unit := range g.Self(enemy).Units {
			g.Action(enemy, "set-unit-behavior", gin.H{"internal_ids": []uint64{unit.InternalID}, "stance": uint8(defs.UnitStance_PASSIVE), "attack_back": false})
		}

		g.Action(owner, "set-unit-behavior", gin.H{
			"internal_ids":      []uint64{attacker.InternalID},
			"stance":            uint8(defs.UnitStance_AGGRESSIVE),
			"attack_back":       false,
			"interrupt_current": false,
			"target_priority":   []uint8{uint8(defs.TargetCategory_MILITARY)},
		})

		g.Submit(owner)
		g.Submit(enemy)

		u := g.Self(owner).Units[0]
		hasAutoAttack := false
		for _, o := range u.ActiveOrders {
			if defs.OrderType(o.OrderType).IsAttackOrder() {
				hasAutoAttack = true
				break
			}
		}
		villager := g.Self(enemy).Units[0]
		building := g.Self(enemy).Buildings[0]
		damageDealt := villager.CombatStats.Health < float64(villager.CombatStats.MaxHealth) || building.CombatStats.Health < float64(building.CombatStats.MaxHealth)
		if !hasAutoAttack && !damageDealt {
			t.Fatalf("expected idle attacker to acquire unprioritized target via fallback, got orders: %+v", u.ActiveOrders)
		}
	})
}
