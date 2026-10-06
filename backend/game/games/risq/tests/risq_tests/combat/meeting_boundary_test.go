package combat

import (
	"fmt"
	"github.com/dgray001/gray_online/game/games/risq/tests/harness"
	"testing"
)

func TestMeetingStaminaBoundaries(t *testing.T) {
	for _, c := range []struct {
		cost                  uint
		stamina, sunk, attack int
	}{
		{1, 15, 1, 2}, {2, 15, 1, 2}, {3, 15, 2, 1}, {4, 15, 2, 1}, {5, 15, 3, 0}, {6, 15, 3, 0}, {9, 15, 5, 0}, {12, 15, 6, 0}, {1, 2, 1, 1}, {3, 3, 2, 1}, {1, 1, 1, 0},
	} {
		t.Run(fmt.Sprintf("cost-%d-stamina-%d", c.cost, c.stamina), func(t *testing.T) {
			log := captureTicks(t)
			g := meetingBoundaryGame(t, c.cost, c.stamina, 15)
			a, b := soleUnit(g, 0), soleUnit(g, 1)
			makePassive(g, 1, b)
			g.Submit(g.Human(0), attackOrder(a, b))
			g.Submit(g.Human(1), harness.OrderMoveZone([]uint64{b.InternalID}, 0, 0, 0, 0))
			ticks := parseTicks(t, log)
			meeting, found := ticks.meetingOf(int(a.InternalID))
			if !found || meeting.tick != 1 || meeting.sunk != c.sunk || meeting.attack != c.attack {
				t.Fatalf("meeting=%+v found=%v, want sunk %d attack %d", meeting, found, c.sunk, c.attack)
			}
			hits := ticks.hitsBy(int(a.InternalID))
			firstTickAttack := len(hits) > 0 && hits[0].tick == 1
			if firstTickAttack != (c.attack > 0) {
				t.Fatalf("first tick hits=%+v", hits)
			}
			if firstTickAttack && hits[0].stamina != c.attack {
				t.Fatalf("attack stamina=%d, want %d", hits[0].stamina, c.attack)
			}
		})
	}
}
