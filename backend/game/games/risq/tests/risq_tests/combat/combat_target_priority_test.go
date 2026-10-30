package combat

import (
	"fmt"
	"testing"

	"github.com/dgray001/gray_online/game/games/risq/internal/defs"
	"github.com/dgray001/gray_online/game/games/risq/tests/harness"
	"github.com/dgray001/gray_online/game/games/risq/tests/harness/fakeboard"
	"github.com/gin-gonic/gin"
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

func TestIdleAutoAttackRequiresListedCategory(t *testing.T) {
	for _, priority := range [][]uint8{{}, {uint8(defs.TargetCategory_BUILDING)}} {
		g := stanceGame(t, sameZone)
		owner, enemy := g.Human(0), g.Human(1)
		attacker, target := g.Self(owner).Units[0], g.Self(enemy).Units[0]
		g.Action(owner, "set-unit-behavior", gin.H{"internal_ids": []uint64{attacker.InternalID}, "stance": uint8(defs.UnitStance_AGGRESSIVE), "attack_back": false, "target_priority": priority})
		passive(g, enemy, target.InternalID)
		g.EndTurn()
		if len(g.Self(owner).Units[0].ActiveOrders) != 0 || g.Self(enemy).Units[0].CombatStats.Health != target.CombatStats.Health {
			t.Fatalf("idle unit attacked an unlisted category with priority %v", priority)
		}
	}
}

func TestBuildingAutoAttackRequiresListedCategory(t *testing.T) {
	for _, priority := range [][]uint8{{}, {uint8(defs.TargetCategory_ECONOMIC)}} {
		fakeboard.UseConfig(t, testConfig, nil, map[string]string{"priority-tower": fmt.Sprintf(meetingBoard, towerSpaces(center, sameWest))})
		g := harness.NewGame(t, "custom:priority-tower", 1, 2)
		owner, enemy := g.Human(0), g.Human(1)
		tower, target := g.Self(owner).Buildings[0], g.Self(enemy).Units[0]
		g.Action(owner, "set-building-behavior", gin.H{"internal_ids": []uint64{tower.InternalID}, "auto_attack": true, "target_priority": priority})
		passive(g, enemy, target.InternalID)
		g.EndTurn()
		if len(g.Self(owner).ActiveOrders) != 0 || g.Self(enemy).Units[0].CombatStats.Health != target.CombatStats.Health {
			t.Fatalf("building attacked an unlisted category with priority %v", priority)
		}
	}
}

func TestTargetPriority(t *testing.T) {
	doc := `{"board_size":1,"players":2,"spaces":[{"x":0,"y":0,"terrain":1,"zones":[{"x":0,"y":0,"units":[{"id":22,"player":0,"count":1},{"id":11,"player":1,"count":10},{"id":21,"player":1,"count":1}]}]}]}`
	fakeboard.UseConfig(t, testConfig, nil, map[string]string{"combat_priority": doc})
	g := harness.NewGame(t, "custom:combat_priority", 1, 2)
	p0, p1 := g.Human(0), g.Human(1)
	u0 := g.Self(p0).Units[0]

	// Set target priority to prefer ECONOMIC over GRUNT
	g.Action(p0, "set-unit-behavior", gin.H{
		"internal_ids":    []uint64{u0.InternalID},
		"target_priority": []uint8{uint8(defs.TargetCategory_ECONOMIC), uint8(defs.TargetCategory_MILITARY)},
	})

	// Order attack on space
	g.Submit(p0, harness.Order(defs.OrderType_UnitAttackSpace, []uint64{u0.InternalID}, harness.SpaceKey(0, 0), false))
	g.Submit(p1)

	var grunt *harness.Unit
	survivors := g.Self(p1).Units
	for i := range survivors {
		if survivors[i].UnitID == 21 {
			grunt = &survivors[i]
		}
	}
	if grunt == nil {
		t.Errorf("grunt should have survived, but it was killed")
	} else if grunt.CombatStats.Health < 30 {
		t.Errorf("grunt incorrectly took damage, health %v", grunt.CombatStats.Health)
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

	t.Run("idle unit does not attack with empty priority", func(t *testing.T) {
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
			"target_priority":   []uint8{},
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
		worker := g.Self(enemy).Units[0]
		building := g.Self(enemy).Buildings[0]
		damageDealt := worker.CombatStats.Health < float64(worker.CombatStats.MaxHealth) || building.CombatStats.Health < float64(building.CombatStats.MaxHealth)
		if hasAutoAttack || damageDealt {
			t.Fatalf("idle attacker attacked an unprioritized target, got orders: %+v", u.ActiveOrders)
		}
	})
}
