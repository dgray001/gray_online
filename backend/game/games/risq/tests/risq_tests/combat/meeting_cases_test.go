package combat

import (
	"fmt"
	"reflect"
	"sort"
	"testing"

	"github.com/dgray001/gray_online/game/games/risq/internal/defs"
	"github.com/dgray001/gray_online/game/games/risq/tests/harness"
)

func unitsOf(g *harness.Game, slot int) []harness.Unit {
	units := g.Self(g.Human(slot)).Units
	sort.Slice(units, func(i, j int) bool { return units[i].InternalID < units[j].InternalID })
	return units
}

func moveTo(unit harness.Unit, x, y, zx, zy int) defs.OrderFromFrontend {
	return harness.OrderMoveZone([]uint64{unit.InternalID}, x, y, zx, zy)
}

func TestMeetingStationaryTargetIsApproachedNormally(t *testing.T) {
	log := captureTicks(t)
	g := meetingGame(t, spaceOf(0, 0, flat, placed{heavy, 0, 0, 0}, placed{heavy, 1, 1, 0}))
	a, b := soleUnit(g, 0), soleUnit(g, 1)
	makePassive(g, 1, b)
	g.Submit(g.Human(0), attackOrder(a, b))
	g.Submit(g.Human(1))
	ticks := parseTicks(t, log)

	if len(ticks.meetings) != 0 {
		t.Errorf("meetings %+v, want none against a unit that is not crossing", ticks.meetings)
	}
	hits := ticks.hitsBy(int(a.InternalID))
	if moves := ticks.movesOf(int(a.InternalID)); len(moves) != 1 || moves[0].tick != 1 || len(hits) == 0 || hits[0].tick != 2 || hits[0].stamina != 3 {
		t.Errorf("moves %+v hits %+v, want one step in tick 1 and a full hit from tick 2", moves, hits)
	}
}

func TestMeetingTargetStepsElsewhere(t *testing.T) {
	log := captureTicks(t)
	g := meetingGame(t, spaceOf(0, 0, flat, placed{heavy, 0, 0, 0}, placed{heavy, 1, 1, 0}))
	a, b := soleUnit(g, 0), soleUnit(g, 1)
	makePassive(g, 1, b)
	g.Submit(g.Human(0), attackOrder(a, b))
	g.Submit(g.Human(1), moveTo(b, 0, 0, 1, -1))
	ticks := parseTicks(t, log)

	if len(ticks.meetings) != 0 {
		t.Errorf("meetings %+v, want none when B does not step into A's zone", ticks.meetings)
	}
	if moves := ticks.movesOf(int(a.InternalID)); len(moves) == 0 || moves[0].tick != 1 {
		t.Errorf("A moves %+v, want it to step toward B's old zone in tick 1", moves)
	}
}

func TestMeetingEveryChaserInTheZoneMeets(t *testing.T) {
	log := captureTicks(t)
	g := meetingGame(t, spaceOf(0, 0, flat, placed{heavy, 0, 1, 0}, placed{heavy, 0, 1, 0}, placed{heavy, 1, 0, 0}))
	chasers, b := unitsOf(g, 0), soleUnit(g, 1)
	makePassive(g, 1, b)
	g.Submit(g.Human(0), attackOrder(chasers[0], b), attackOrder(chasers[1], b))
	g.Submit(g.Human(1), moveTo(b, 0, 0, 1, 0))
	ticks := parseTicks(t, log)

	for _, chaser := range chasers {
		m, ok := ticks.meetingOf(int(chaser.InternalID))
		hits := ticks.hitsBy(int(chaser.InternalID))
		if !ok || m.tick != 1 || len(hits) == 0 || hits[0].tick != 1 || hits[0].stamina != 2 {
			t.Errorf("chaser %d: meeting %+v (found %v) hits %+v, want a tick 1 meeting and a 2-stamina hit", chaser.InternalID, m, ok, hits)
		}
	}
}

// B stays in its zone to meet C, yet A still meets B at the border because B intended to cross when the tick was decided
func TestMeetingDecidesFromOriginalIntents(t *testing.T) {
	// units are processed by player slot, so each side owning B makes it the first or the last one handled
	for bSlot := 0; bSlot <= 1; bSlot++ {
		t.Run(fmt.Sprintf("B in slot %d", bSlot), func(t *testing.T) {
			log := captureTicks(t)
			other := 1 - bSlot
			g := meetingGame(t, spaceOf(0, 0, flat, placed{heavy, other, 1, 0}, placed{heavy, other, 1, 0}, placed{heavy, bSlot, 0, 0}))
			mine, b := unitsOf(g, other), soleUnit(g, bSlot)
			a, c := mine[0], mine[1]
			g.Submit(g.Human(other), attackOrder(a, b), moveTo(c, 0, 0, 0, 0))
			g.Submit(g.Human(bSlot), attackOrder(b, c))
			ticks := parseTicks(t, log)

			for name, unit := range map[string]harness.Unit{"A": a, "B": b} {
				m, ok := ticks.meetingOf(int(unit.InternalID))
				hits := ticks.hitsBy(int(unit.InternalID))
				if !ok || m.tick != 1 || len(hits) == 0 || hits[0].tick != 1 || hits[0].stamina != 2 {
					t.Errorf("%s: meeting %+v (found %v) hits %+v, want a tick 1 meeting and a 2-stamina hit", name, m, ok, hits)
				}
				if moves := ticks.movesOf(int(unit.InternalID)); len(moves) != 0 && moves[0].tick == 1 {
					t.Errorf("%s moved in tick 1, want it to stay in its zone", name)
				}
			}
			if moves := ticks.movesOf(int(c.InternalID)); len(moves) == 0 || moves[0].tick != 1 {
				t.Errorf("C moves %+v, want its plain step into B's zone in tick 1", moves)
			}
		})
	}
}

// A's hits in the one-sided scenario with A or B placed first (so their ids swap order) and either in player slot 0
func meetingHits(t *testing.T, attackerSlot int, attackerFirst bool) [][3]float64 {
	log := captureTicks(t)
	a, b := placed{heavy, attackerSlot, 0, 0}, placed{heavy, 1 - attackerSlot, 1, 0}
	units := []placed{a, b}
	if !attackerFirst {
		units = []placed{b, a}
	}
	g := meetingGame(t, spaceOf(0, 0, flat, units...))
	au, bu := soleUnit(g, attackerSlot), soleUnit(g, 1-attackerSlot)
	g.Submit(g.Human(attackerSlot), attackOrder(au, bu))
	g.Submit(g.Human(1-attackerSlot), moveTo(bu, 0, 0, 0, 0))
	var out [][3]float64
	for _, h := range parseTicks(t, log).hitsBy(int(au.InternalID)) {
		out = append(out, [3]float64{float64(h.tick), float64(h.stamina), h.damage})
	}
	return out
}

func TestMeetingIgnoresUnitIdsAndPlayerSlots(t *testing.T) {
	want := meetingHits(t, 0, true)
	if len(want) < 2 || want[0][0] != 1 || want[0][1] != 2 {
		t.Fatalf("baseline hits %v, want a 2-stamina hit in tick 1 followed by more", want)
	}
	for name, got := range map[string][][3]float64{
		"attacker placed second": meetingHits(t, 0, false),
		"attacker in slot 1":     meetingHits(t, 1, true),
		"slot 1 and second":      meetingHits(t, 1, false),
	} {
		if !reflect.DeepEqual(got, want) {
			t.Errorf("%s: hits %v, want %v", name, got, want)
		}
	}
}
