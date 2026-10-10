package combat

import (
	"github.com/dgray001/gray_online/game/games/risq/internal/defs"
	"testing"
)

func TestPositiveRangeShootsAcrossBorder(t *testing.T) {
	for name, attackRange := range map[string]defs.RisqRange{"adjacent": defs.RisqRange_ADJACENT, "secondary": defs.RisqRange_SECONDARY} {
		t.Run(name, func(t *testing.T) {
			log := captureTicks(t)
			g := spaceRangeGame(t, attackRange, acrossSpaces(placed{heavy, 0, 1, 0}, placed{heavy, 1, -1, 0})...)
			a, b := soleUnit(g, 0), soleUnit(g, 1)
			g.Submit(g.Human(0), attackOrder(a, b))
			g.Submit(g.Human(1), attackOrder(b, a))
			ticks := parseTicks(t, log)
			if len(ticks.moves) != 0 || len(ticks.meetings) != 0 || len(ticks.hitsBy(int(a.InternalID))) == 0 || len(ticks.hitsBy(int(b.InternalID))) == 0 {
				t.Fatalf("positive range did not shoot normally: %+v", ticks)
			}
		})
	}
}
