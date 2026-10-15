package combat

import (
	"fmt"
	"testing"

	"github.com/dgray001/gray_online/game/games/risq/internal/defs"
	"github.com/dgray001/gray_online/game/games/risq/tests/harness"
	"github.com/gin-gonic/gin"
)

func checkCrossingMoves(t *testing.T, cross bool, stance defs.UnitStance, interrupt, prioritized bool) {
	log := captureTicks(t)
	spaces := []string{spaceOf(0, 0, flat, placed{heavy, 0, 0, 0}, placed{heavy, 1, 1, 0})}
	if cross {
		spaces = []string{spaceOf(0, 0, flat, placed{heavy, 0, 1, 0}), spaceOf(1, 0, flat, placed{heavy, 1, -1, 0})}
	}
	g := meetingGame(t, spaces...)
	a, b := soleUnit(g, 0), soleUnit(g, 1)
	priority := []uint8{}
	if prioritized {
		priority = []uint8{uint8(defs.TargetCategory_MILITARY)}
	}
	for _, u := range []harness.Unit{a, b} {
		g.Action(g.Human(u.PlayerID), "set-unit-behavior", gin.H{"internal_ids": []uint64{u.InternalID}, "stance": uint8(stance), "interrupt_current": interrupt, "attack_back": false, "target_priority": priority})
	}
	g.Submit(g.Human(0), moveTo(a, b.Space.X, b.Space.Y, b.Zone.X, b.Zone.Y))
	g.Submit(g.Human(1), moveTo(b, a.Space.X, a.Space.Y, a.Zone.X, a.Zone.Y))
	checkCrossingOutcome(t, g, a, b, parseTicks(t, log), interrupt && prioritized && stance != defs.UnitStance_PASSIVE)
}

func checkCrossingOutcome(t *testing.T, g *harness.Game, a, b harness.Unit, ticks tickLog, fight bool) {
	ra, rb := harness.RequireReplay(t, g.State(g.Human(0))), harness.RequireReplay(t, g.State(g.Human(1)))
	aa, bb := ra.UnitAt(a.InternalID, 1), rb.UnitAt(b.InternalID, 1)
	if aa == nil || bb == nil {
		t.Fatal("missing tick-one units")
	}
	swapped := aa.Space == b.Space && aa.Zone == b.Zone && bb.Space == a.Space && bb.Zone == a.Zone
	if swapped == fight || (len(ticks.hits) > 0) != fight || (len(ticks.meetings) > 0) != fight {
		t.Fatalf("fight=%t swapped=%t hits=%v meetings=%v", fight, swapped, ticks.hits, ticks.meetings)
	}
	if fight && (aa.Space != a.Space || aa.Zone != a.Zone || bb.Space != b.Space || bb.Zone != b.Zone) {
		t.Fatal("crossing combatants moved during their meeting")
	}
}

func TestMeetingInterruptsCrossingMoves(t *testing.T) {
	for _, cross := range []bool{false, true} {
		for _, stance := range []defs.UnitStance{defs.UnitStance_PASSIVE, defs.UnitStance_AGGRESSIVE, defs.UnitStance_DEFENSIVE, defs.UnitStance_STAND_GROUND} {
			for _, interrupt := range []bool{false, true} {
				for _, prioritized := range []bool{false, true} {
					t.Run(fmt.Sprintf("cross=%t/stance=%d/interrupt=%t/prioritized=%t", cross, stance, interrupt, prioritized), func(t *testing.T) {
						checkCrossingMoves(t, cross, stance, interrupt, prioritized)
					})
				}
			}
		}
	}
}

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
