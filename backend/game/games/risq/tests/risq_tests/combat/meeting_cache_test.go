package combat

import (
	"reflect"
	"testing"
)

func acrossSpaces(a, b placed) []string {
	return []string{spaceOf(0, 0, flat, a), spaceOf(1, 0, flat, b)}
}

// 15 stamina: 3 sunk once, then 3-stamina hits in ticks 2 to 5 until the last 3 stamina no longer leaves the rest of the step
func TestMeetingCachedHalfMoveIsPaidOncePerTurn(t *testing.T) {
	log := captureTicks(t)
	g := meetingGame(t, acrossSpaces(placed{blunt, 0, 1, 0}, placed{blunt, 1, -1, 0})...)
	a, b := soleUnit(g, 0), soleUnit(g, 1)
	g.Submit(g.Human(0), attackOrder(a, b))
	g.Submit(g.Human(1), attackOrder(b, a))
	ticks := parseTicks(t, log)

	for _, unit := range []int{int(a.InternalID), int(b.InternalID)} {
		hits := ticks.hitsBy(unit)
		if got := staminaOf(hits); !reflect.DeepEqual(got, []int{3, 3, 3, 3}) || hits[0].tick != 2 || hits[3].tick != 5 {
			t.Errorf("unit %d hits %+v, want four full hits in ticks 2 to 5", unit, hits)
		}
		sunk := 0
		for _, m := range ticks.meetings {
			if m.unit == unit {
				sunk += m.sunk
			}
		}
		if sunk != 3 {
			t.Errorf("unit %d paid %d sunk in total, want the half move of 3 once", unit, sunk)
		}
	}
	if len(ticks.moves) != 0 {
		t.Errorf("moves %+v, want both to stay put", ticks.moves)
	}
}

// B runs out of stamina, so A's step becomes real: A pays only the rest of the step (6 in all) and ends in B's zone
func TestMeetingCachedHalfMoveCountsTowardARealStep(t *testing.T) {
	log := captureTicks(t)
	g := meetingGame(t, acrossSpaces(placed{blunt, 0, 1, 0}, placed{heavy, 1, -1, 0})...)
	a, b := soleUnit(g, 0), soleUnit(g, 1)
	g.Submit(g.Human(0), attackOrder(a, b))
	g.Submit(g.Human(1), attackOrder(b, a))
	ticks := parseTicks(t, log)

	var meetings []meetingLine
	for _, m := range ticks.meetings {
		if m.unit == int(a.InternalID) {
			meetings = append(meetings, m)
		}
	}
	if len(meetings) != 2 || meetings[0].sunk != 3 || meetings[1].sunk != 0 || meetings[1].tick != 2 {
		t.Errorf("A's meetings %+v, want tick 1 with 3 sunk and tick 2 with none", meetings)
	}
	if moves := ticks.movesOf(int(a.InternalID)); len(moves) != 1 || moves[0].tick != 3 || moves[0].cost != 3 {
		t.Errorf("A's moves %+v, want one step in tick 3 costing the remaining 3", moves)
	}
	if after := soleUnit(g, 0); after.Zone != b.Zone || after.Space != b.Space {
		t.Errorf("A ended at %v/%v, want B's zone %v/%v", after.Space, after.Zone, b.Space, b.Zone)
	}
}

// The cache lives for one turn: the next turn's first meeting pays the half move again
func TestMeetingHalfMoveIsPaidAgainNextTurn(t *testing.T) {
	log := captureTicks(t)
	g := meetingGame(t, acrossSpaces(placed{heavy, 0, 1, 0}, placed{heavy, 1, -1, 0})...)
	a, b := soleUnit(g, 0), soleUnit(g, 1)
	g.Submit(g.Human(0), attackOrder(a, b))
	g.Submit(g.Human(1), attackOrder(b, a))
	g.EndTurn()
	ticks := parseTicks(t, log)

	sunkByTurn := map[int][]int{}
	for _, m := range ticks.meetings {
		if m.unit == int(a.InternalID) {
			sunkByTurn[m.turn] = append(sunkByTurn[m.turn], m.sunk)
		}
	}
	if len(sunkByTurn) != 2 {
		t.Fatalf("A met in turns %v, want two turns", sunkByTurn)
	}
	for turn, sunk := range sunkByTurn {
		if len(sunk) == 0 || sunk[0] != 3 {
			t.Errorf("turn %d meeting sunk %v, want the half move of 3 paid at the first meeting", turn, sunk)
		}
		for _, later := range sunk[1:] {
			if later != 0 {
				t.Errorf("turn %d sunk %v, want nothing more after the first meeting", turn, sunk)
			}
		}
	}
}
