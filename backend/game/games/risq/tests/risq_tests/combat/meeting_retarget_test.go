package combat

import (
	"github.com/dgray001/gray_online/game/games/risq/internal/defs"
	"github.com/dgray001/gray_online/game/games/risq/tests/harness"
	"github.com/gin-gonic/gin"
	"testing"
)

func TestMeetingCacheDoesNotDiscountAnotherZone(t *testing.T) {
	log := captureTicks(t)
	g := meetingGame(t, spaceOf(0, 0, flat, placed{blunt, 0, 1, 0}), spaceOf(1, 0, flat, placed{heavy, 1, -1, 0}), spaceOf(0, 1, flat, placed{heavy, 1, 0, -1}))
	a := soleUnit(g, 0)
	var crossing, arriving harness.Unit
	for _, unit := range g.Self(g.Human(1)).Units {
		if unit.Space.X == 1 {
			crossing = unit
		} else {
			arriving = unit
		}
		makePassive(g, 1, unit)
	}
	g.Action(g.Human(0), "set-unit-behavior", gin.H{"internal_ids": []uint64{a.InternalID}, "stance": uint8(defs.UnitStance_AGGRESSIVE), "interrupt_current": true, "attack_back": false})
	g.Submit(g.Human(0))
	g.Submit(g.Human(1), attackOrder(crossing, a), harness.OrderMoveZone([]uint64{arriving.InternalID}, 0, 0, 0, 1))
	ticks := parseTicks(t, log)
	meeting, found := ticks.meetingOf(int(a.InternalID))
	if !found || meeting.tick != 1 || meeting.sunk != 3 {
		t.Fatalf("meeting=%+v found=%v", meeting, found)
	}
	moves := ticks.movesOf(int(a.InternalID))
	if len(moves) == 0 || moves[0].tick != 2 || moves[0].cost != 1 {
		t.Fatalf("retargeted moves=%+v, want full-cost intra-space step", moves)
	}
}
