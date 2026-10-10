package combat

import (
	"github.com/dgray001/gray_online/game/games/risq/internal/defs"
	"github.com/dgray001/gray_online/game/games/risq/tests/harness"
	"testing"
)

func TestSpaceRangeShootsWithinSpace(t *testing.T) {
	for name, kind := range map[string]defs.OrderType{"unit": defs.OrderType_UnitAttackUnit, "zone": defs.OrderType_UnitAttackZone, "space": defs.OrderType_UnitAttackSpace} {
		t.Run(name, func(t *testing.T) {
			log := captureTicks(t)
			g := spaceRangeGame(t, defs.RisqRange_SPACE, spaceOf(0, 0, flat, placed{heavy, 0, 0, 0}, placed{heavy, 1, 1, 0}))
			a, b := soleUnit(g, 0), soleUnit(g, 1)
			g.Submit(g.Human(0), spaceRangeAttackOrder(a, b, kind))
			g.Submit(g.Human(1), spaceRangeAttackOrder(b, a, kind))
			ticks := parseTicks(t, log)
			if len(ticks.moves) != 0 || len(ticks.meetings) != 0 {
				t.Fatalf("same-space shooting moved or met: %+v", ticks)
			}
			for _, unit := range []harness.Unit{a, b} {
				hits := ticks.hitsBy(int(unit.InternalID))
				if len(hits) == 0 || hits[0].tick != 1 || hits[0].stamina != 3 {
					t.Errorf("unit %d attacks %+v, want full-strength shooting on tick 1", unit.InternalID, hits)
				}
			}
		})
	}
}
