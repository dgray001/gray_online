package combat

import (
	"math"
	"testing"

	"github.com/dgray001/gray_online/game/games/risq/tests/harness"
)

// A at the center zone chases B in the adjacent zone, while B steps into A's zone without attacking A
func TestMeetingIntraSpaceOneSided(t *testing.T) {
	log := captureTicks(t)
	g := meetingGame(t, spaceOf(0, 0, flat, placed{heavy, 0, 0, 0}, placed{heavy, 1, 1, 0}))
	a, b := soleUnit(g, 0), soleUnit(g, 1)
	g.Submit(g.Human(0), attackOrder(a, b))
	g.Submit(g.Human(1), harness.OrderMoveZone([]uint64{b.InternalID}, 0, 0, 0, 0))
	ticks := parseTicks(t, log)

	meeting, ok := ticks.meetingOf(int(a.InternalID))
	if !ok || meeting.tick != 1 || meeting.sunk != 1 || meeting.attack != 2 {
		t.Fatalf("meeting %+v (found %v), want tick 1 with 1 stamina sunk and a 2-stamina attack", meeting, ok)
	}
	if moves := ticks.movesOf(int(a.InternalID)); len(moves) != 0 {
		t.Errorf("A moved %v, want it to stay in its zone", moves)
	}
	if moves := ticks.movesOf(int(b.InternalID)); len(moves) != 1 || moves[0].tick != 1 {
		t.Errorf("B moves %v, want exactly one step in tick 1", moves)
	}
	hits := ticks.hitsBy(int(a.InternalID))
	if len(hits) < 2 || hits[0].tick != 1 || hits[0].stamina != 2 || hits[1].tick != 2 || hits[1].stamina != 3 {
		t.Fatalf("A's hits %+v, want a 2-stamina hit in tick 1 and a full 3-stamina hit in tick 2", hits)
	}
	if ratio := hits[0].damage / hits[1].damage; math.Abs(ratio-2.0/3.0) > 1e-3 {
		t.Errorf("tick 1 damage is %.4f of a full hit, want 2/3", ratio)
	}
	if after := soleUnit(g, 1); after.Zone != (harness.Coord{}) {
		t.Errorf("B ended in zone %v, want it to have arrived in A's zone (0,0)", after.Zone)
	}
}
