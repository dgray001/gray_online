package combat

import (
	"reflect"
	"testing"

	"github.com/dgray001/gray_online/game/games/risq/tests/harness"
)

func staminaOf(hits []hitLine) []int {
	out := make([]int, len(hits))
	for i, h := range hits {
		out[i] = h.stamina
	}
	return out
}

func TestTickHistoryMeleeReplacement(t *testing.T) {
	log := captureTicks(t)
	g := meetingGame(t, spaceOf(0, 0, flat, placed{heavy, 0, 0, 0}, placed{heavy, 1, 1, 0}))
	a, b := soleUnit(g, 0), soleUnit(g, 1)
	g.Submit(g.Human(0), attackOrder(a, b))
	g.Submit(g.Human(1), attackOrder(b, a))
	if len(parseTicks(t, log).meetings) != 6 {
		t.Fatal("fixture did not meet for three ticks")
	}
	for _, u := range []harness.Unit{a, b} {
		actions := g.State(g.Human(u.PlayerID)).Unit(u.InternalID).TickActions
		first := harness.RequireTickAction(t, actions, "move")
		if first.Tick != 1 || first.Intent.Resolution.Kind != "attack" || first.Intent.Resolution.SunkCost != 1 || first.Execute.StaminaSpent != 3 {
			t.Errorf("lost original move or melee replacement: %+v", first)
		}
		harness.AssertTickSpend(t, actions, u.CurrentStamina)
		for _, hit := range parseTicks(t, log).hitsBy(int(u.InternalID)) {
			matched := false
			for _, action := range actions {
				if action.Tick == hit.tick && action.Execute.Target.InternalID == uint64(hit.target) {
					matched = true
				}
			}
			if !matched {
				t.Errorf("missing executed melee attack for %+v", hit)
			}
		}
	}
}

// Both chase each other, so both meet at the border every tick and neither ever steps into the other's zone
func TestMeetingMutualIntraSpaceClashesEveryTick(t *testing.T) {
	log := captureTicks(t)
	g := meetingGame(t, spaceOf(0, 0, flat, placed{heavy, 0, 0, 0}, placed{heavy, 1, 1, 0}))
	a, b := soleUnit(g, 0), soleUnit(g, 1)
	g.Submit(g.Human(0), attackOrder(a, b))
	g.Submit(g.Human(1), attackOrder(b, a))
	ticks := parseTicks(t, log)

	// 8 stamina: tick 1 pays the half move of 1 and attacks with 2, then full 3-stamina hits until the last 2 stamina
	want := []int{2, 3, 2}
	for _, unit := range []harness.Unit{a, b} {
		if got := staminaOf(ticks.hitsBy(int(unit.InternalID))); !reflect.DeepEqual(got, want) {
			t.Errorf("unit %d hit with %v stamina, want %v", unit.InternalID, got, want)
		}
		if moves := ticks.movesOf(int(unit.InternalID)); len(moves) != 0 {
			t.Errorf("unit %d moved %v, want both to hold their zones", unit.InternalID, moves)
		}
	}
	if len(ticks.meetings) != 6 {
		t.Errorf("%d meetings, want 3 ticks for each of 2 units: %+v", len(ticks.meetings), ticks.meetings)
	}
	after0, after1 := soleUnit(g, 0), soleUnit(g, 1)
	if after0.Zone != a.Zone || after1.Zone != b.Zone {
		t.Errorf("zones moved to %v and %v, want %v and %v", after0.Zone, after1.Zone, a.Zone, b.Zone)
	}
	if after0.CombatStats.Health != after1.CombatStats.Health || after0.CombatStats.Health >= a.CombatStats.Health {
		t.Errorf("health %v and %v, want both hurt by the same amount", after0.CombatStats.Health, after1.CombatStats.Health)
	}
}

// The 6-stamina crossing halves to 3, the whole tick's budget, so tick 1 only pays it; later ticks hit at full rate until
// the stamina left (2) cannot cover the remaining 3 of the step, so the unit stops with it unspent
func TestMeetingMutualAcrossSpacesPaysTheHalfMoveOnce(t *testing.T) {
	log := captureTicks(t)
	g := meetingGame(t, spaceOf(0, 0, flat, placed{heavy, 0, 1, 0}), spaceOf(1, 0, flat, placed{heavy, 1, -1, 0}))
	a, b := soleUnit(g, 0), soleUnit(g, 1)
	g.Submit(g.Human(0), attackOrder(a, b))
	g.Submit(g.Human(1), attackOrder(b, a))
	ticks := parseTicks(t, log)

	for _, unit := range []harness.Unit{a, b} {
		id := int(unit.InternalID)
		var got []meetingLine
		for _, m := range ticks.meetings {
			if m.unit == id {
				got = append(got, m)
			}
		}
		if len(got) != 2 || got[0].tick != 1 || got[0].sunk != 3 || got[0].attack != 0 || got[1].tick != 2 || got[1].sunk != 0 || got[1].attack != 3 {
			t.Errorf("unit %d meetings %+v, want tick 1 sunk 3 attack 0, then tick 2 sunk 0 attack 3 and no more", id, got)
		}
		if hits := ticks.hitsBy(id); len(hits) != 1 || hits[0].tick != 2 || hits[0].stamina != 3 {
			t.Errorf("unit %d hits %+v, want a single full hit in tick 2", id, hits)
		}
	}
	if len(ticks.moves) != 0 {
		t.Errorf("moves %+v, want both to hold their zones", ticks.moves)
	}
	if after0, after1 := soleUnit(g, 0), soleUnit(g, 1); after0.Zone != a.Zone || after1.Zone != b.Zone || after0.CombatStats.Health != after1.CombatStats.Health || after0.CombatStats.Health >= a.CombatStats.Health {
		t.Errorf("zones %v %v health %v %v, want zones kept and both hurt equally", after0.Zone, after1.Zone, after0.CombatStats.Health, after1.CombatStats.Health)
	}
}

// Only an attack order meets at the border; two plain movers still swap zones
func TestMeetingNeedsAnAttackOrder(t *testing.T) {
	log := captureTicks(t)
	g := meetingGame(t, spaceOf(0, 0, flat, placed{heavy, 0, 0, 0}, placed{heavy, 1, 1, 0}))
	a, b := soleUnit(g, 0), soleUnit(g, 1)
	g.Submit(g.Human(0), harness.OrderMoveZone([]uint64{a.InternalID}, 0, 0, 1, 0))
	g.Submit(g.Human(1), harness.OrderMoveZone([]uint64{b.InternalID}, 0, 0, 0, 0))
	ticks := parseTicks(t, log)

	if len(ticks.meetings) != 0 {
		t.Errorf("meetings %+v, want none for plain moves", ticks.meetings)
	}
	if after0, after1 := soleUnit(g, 0), soleUnit(g, 1); after0.Zone != b.Zone || after1.Zone != a.Zone {
		t.Errorf("zones %v and %v, want the two to have swapped to %v and %v", after0.Zone, after1.Zone, b.Zone, a.Zone)
	}
}
