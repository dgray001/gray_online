package combat

import (
	"github.com/dgray001/gray_online/game/games/risq/tests/harness"
	"testing"
)

func TestMeetingCacheClearsAfterAttackingInPlace(t *testing.T) {
	log := captureTicks(t)
	g := meetingBoundaryGame(t, 6, 15, 15)
	a, b := soleUnit(g, 0), soleUnit(g, 1)
	makePassive(g, 1, b)
	g.Submit(g.Human(0), attackOrder(a, b))
	g.Submit(g.Human(1), harness.OrderMoveZone([]uint64{b.InternalID}, 0, 0, 0, 0), harness.OrderMoveZone([]uint64{b.InternalID}, 0, 0, 1, 0))
	ticks := parseTicks(t, log)
	meeting, found := ticks.meetingOf(int(a.InternalID))
	if !found || meeting.sunk != 3 {
		t.Fatalf("initial meeting=%+v", meeting)
	}
	moves, hits := ticks.movesOf(int(a.InternalID)), ticks.hitsBy(int(a.InternalID))
	if len(moves) != 1 || moves[0].cost != 6 || len(hits) == 0 || hits[0].tick >= moves[0].tick {
		t.Fatalf("moves=%+v hits=%+v, want full-cost step after attack", moves, hits)
	}
}
