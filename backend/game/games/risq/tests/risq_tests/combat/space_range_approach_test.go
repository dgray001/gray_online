package combat

import (
	"github.com/dgray001/gray_online/game/games/risq/internal/defs"
	"github.com/dgray001/gray_online/game/games/risq/tests/harness"
	"testing"
)

func TestSpaceRangeApproachesStationaryEnemy(t *testing.T) {
	for name, kind := range map[string]defs.OrderType{"unit": defs.OrderType_UnitAttackUnit, "zone": defs.OrderType_UnitAttackZone, "space": defs.OrderType_UnitAttackSpace} {
		t.Run(name, func(t *testing.T) {
			log := captureTicks(t)
			g := spaceRangeGame(t, defs.RisqRange_SPACE, acrossSpaces(placed{heavy, 0, 1, 0}, placed{heavy, 1, -1, 0})...)
			a, b := soleUnit(g, 0), soleUnit(g, 1)
			makePassive(g, 1, b)
			g.Submit(g.Human(0), spaceRangeAttackOrder(a, b, kind))
			g.Submit(g.Human(1))
			ticks := parseTicks(t, log)
			if len(ticks.meetings) != 0 || len(ticks.movesOf(int(a.InternalID))) != 1 || len(ticks.hitsBy(int(a.InternalID))) == 0 {
				t.Fatalf("stationary enemy did not get approached and shot: %+v", ticks)
			}
		})
	}
}

func TestSpaceRangeDoesNotMeetNonOpposingMove(t *testing.T) {
	log := captureTicks(t)
	g := spaceRangeGame(t, defs.RisqRange_SPACE, acrossSpaces(placed{heavy, 0, 1, 0}, placed{heavy, 1, -1, 0})...)
	a, b := soleUnit(g, 0), soleUnit(g, 1)
	makePassive(g, 1, b)
	g.Submit(g.Human(0), attackOrder(a, b))
	g.Submit(g.Human(1), harness.OrderMoveZone([]uint64{b.InternalID}, 1, 0, 0, 0))
	ticks := parseTicks(t, log)
	if len(ticks.meetings) != 0 || len(ticks.movesOf(int(a.InternalID))) != 1 || len(ticks.hitsBy(int(a.InternalID))) == 0 {
		t.Fatalf("non-opposing movement incorrectly met or stopped shooting: %+v", ticks)
	}
}
