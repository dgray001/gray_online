package combat

import "testing"

func TestPaidOneStaminaMeetingStepCompletesForFree(t *testing.T) {
	log := captureTicks(t)
	g := meetingBoundaryGame(t, 1, 15, 3)
	a, b := soleUnit(g, 0), soleUnit(g, 1)
	g.Submit(g.Human(0), attackOrder(a, b))
	g.Submit(g.Human(1), attackOrder(b, a))
	moves := parseTicks(t, log).movesOf(int(a.InternalID))
	if len(moves) != 1 || moves[0].tick <= 1 || moves[0].cost != 0 {
		t.Fatalf("moves=%+v, want free completion after meeting", moves)
	}
	if after := soleUnit(g, 0); after.Zone != b.Zone || after.Space != b.Space {
		t.Fatalf("did not complete paid step: %+v", after)
	}
}
