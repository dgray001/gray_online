package combat

import (
	"github.com/dgray001/gray_online/game/games/risq/internal/defs"
	"github.com/dgray001/gray_online/game/games/risq/tests/harness"
	"testing"
)

func TestSpaceRangeCrossingAttacks(t *testing.T) {
	for name, kind := range map[string]defs.OrderType{"unit": defs.OrderType_UnitAttackUnit, "zone": defs.OrderType_UnitAttackZone, "space": defs.OrderType_UnitAttackSpace} {
		t.Run(name, func(t *testing.T) {
			log := captureTicks(t)
			g := spaceRangeGame(t, defs.RisqRange_SPACE, acrossSpaces(placed{heavy, 0, 1, 0}, placed{heavy, 1, -1, 0})...)
			a, b := soleUnit(g, 0), soleUnit(g, 1)
			g.Submit(g.Human(0), spaceRangeAttackOrder(a, b, kind))
			g.Submit(g.Human(1), spaceRangeAttackOrder(b, a, kind))
			ticks := parseTicks(t, log)
			if len(ticks.moves) != 0 {
				t.Fatalf("units crossed instead of meeting: %+v", ticks.moves)
			}
			for slot, before := range []harness.Unit{a, b} {
				after := soleUnit(g, slot)
				if after.Space != before.Space || after.Zone != before.Zone || after.CombatStats.Health >= before.CombatStats.Health {
					t.Errorf("slot %d did not hold position and exchange damage: %+v", slot, after)
				}
				first, found := ticks.meetingOf(int(before.InternalID))
				if !found || first.tick != 1 || first.sunk != 3 || first.attack != 0 {
					t.Errorf("slot %d first meeting %+v found=%v, want a 3-stamina half step", slot, first, found)
				}
			}
		})
	}
}
