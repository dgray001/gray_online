package combat

import (
	"testing"

	"github.com/dgray001/gray_online/game/games/risq/internal/defs"
	"github.com/dgray001/gray_online/game/games/risq/tests/harness"
	"github.com/gin-gonic/gin"
)

const villager = 11

// Attacker on the east edge of (0,0), a villager in the facing zone of (1,0), and a passive phalanx in range in the attacker's own space
func scopeGame(t *testing.T, stance defs.UnitStance, interrupt bool) *harness.Game {
	t.Helper()
	g := spaceRangeGameTweaked(t, defs.RisqRange_SPACE, fullVision,
		spaceOf(0, 0, flat, placed{blunt, 0, 1, 0}, placed{heavy, 1, -1, 0}),
		spaceOf(1, 0, flat, placed{villager, 1, -1, 0}))
	owner, foe := g.Human(0), g.Human(1)
	passive(g, foe, unitWithID(g, foe, heavy).InternalID)
	g.Action(owner, "set-unit-behavior", gin.H{"internal_ids": []uint64{soleUnit(g, 0).InternalID}, "stance": uint8(stance), "attack_back": false, "interrupt_current": interrupt})
	return g
}

func unitWithID(g *harness.Game, human int, unitID uint32) harness.Unit {
	for _, unit := range g.Self(human).Units {
		if unit.UnitID == unitID {
			return unit
		}
	}
	return harness.Unit{}
}

func hurt(unit harness.Unit) bool {
	return unit.CombatStats.Health < float64(unit.CombatStats.MaxHealth)
}

func areaOrder(g *harness.Game, kind defs.OrderType) defs.OrderFromFrontend {
	return spaceRangeAttackOrder(soleUnit(g, 0), unitWithID(g, g.Human(1), villager), kind)
}

func TestAttackAreaFinishesAreaBeforeAnythingElse(t *testing.T) {
	for name, kind := range map[string]defs.OrderType{"space": defs.OrderType_UnitAttackSpace, "zone": defs.OrderType_UnitAttackZone} {
		t.Run(name, func(t *testing.T) {
			g := scopeGame(t, defs.UnitStance_STAND_GROUND, false)
			foe := g.Human(1)
			g.Submit(g.Human(0), areaOrder(g, kind))
			g.Submit(foe)
			if unitWithID(g, foe, villager).InternalID != 0 || hurt(unitWithID(g, foe, heavy)) {
				t.Fatalf("want the area cleared and the in-range phalanx untouched: %+v", g.Self(foe).Units)
			}
			g.EndTurn()
			if !hurt(unitWithID(g, foe, heavy)) {
				t.Fatal("stance did not resume once the area was clear")
			}
		})
	}
}

func TestAttackAreaYieldsToInterruptingStance(t *testing.T) {
	for name, kind := range map[string]defs.OrderType{"space": defs.OrderType_UnitAttackSpace, "zone": defs.OrderType_UnitAttackZone} {
		t.Run(name, func(t *testing.T) {
			g := scopeGame(t, defs.UnitStance_STAND_GROUND, true)
			foe := g.Human(1)
			g.Submit(g.Human(0), areaOrder(g, kind))
			g.Submit(foe)
			if !hurt(unitWithID(g, foe, heavy)) || hurt(unitWithID(g, foe, villager)) {
				t.Fatalf("interrupting stance should take the in-range phalanx over the order: %+v", g.Self(foe).Units)
			}
		})
	}
}
