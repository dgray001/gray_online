package combat

import (
	"testing"

	"github.com/dgray001/gray_online/game/games/risq/internal/defs"
	"github.com/dgray001/gray_online/game/games/risq/tests/harness"
	"github.com/gin-gonic/gin"
)

// B stops reacting to being hit, so only its own step into A's zone matters
func makePassive(g *harness.Game, slot int, unit harness.Unit) {
	g.Action(g.Human(slot), "set-unit-behavior", gin.H{"internal_ids": []uint64{unit.InternalID}, "stance": uint8(defs.UnitStance_PASSIVE), "attack_back": false})
}

func TestMeetingCostFollowsStepCost(t *testing.T) {
	cases := map[string]struct {
		spaces          func() []string
		moveTo          [4]int
		sunk, attack    int
		stepCostSummary string
	}{
		"intra flat, step 1": {func() []string {
			return []string{spaceOf(0, 0, flat, placed{blunt, 0, 0, 0}, placed{heavy, 1, 1, 0})}
		}, [4]int{0, 0, 0, 0}, 1, 2, "half of 1 rounds up to 1"},
		"intra mountains, step 3": {func() []string {
			return []string{spaceOf(0, 0, mountains, placed{blunt, 0, 0, 0}, placed{heavy, 1, 1, 0})}
		}, [4]int{0, 0, 0, 0}, 2, 1, "half of 3 rounds up to 2"},
		"cross flat, step 6": {func() []string {
			return []string{spaceOf(0, 0, flat, placed{blunt, 0, 1, 0}), spaceOf(1, 0, flat, placed{heavy, 1, -1, 0})}
		}, [4]int{0, 0, 1, 0}, 3, 0, "half of 6 uses the whole tick"},
		"cross mountains, step 12": {func() []string {
			return []string{spaceOf(0, 0, flat, placed{blunt, 0, 1, 0}), spaceOf(1, 0, mountains, placed{heavy, 1, -1, 0})}
		}, [4]int{0, 0, 1, 0}, 6, 0, "half of 12 exceeds a tick and is still paid"},
	}
	for name, c := range cases {
		t.Run(name, func(t *testing.T) {
			log := captureTicks(t)
			g := meetingGame(t, c.spaces()...)
			a, b := soleUnit(g, 0), soleUnit(g, 1)
			makePassive(g, 1, b)
			g.Submit(g.Human(0), attackOrder(a, b))
			g.Submit(g.Human(1), harness.OrderMoveZone([]uint64{b.InternalID}, c.moveTo[0], c.moveTo[1], c.moveTo[2], c.moveTo[3]))
			ticks := parseTicks(t, log)

			meeting, ok := ticks.meetingOf(int(a.InternalID))
			if !ok || meeting.tick != 1 || meeting.sunk != c.sunk || meeting.attack != c.attack {
				t.Fatalf("meeting %+v (found %v), want sunk %d attack %d: %s", meeting, ok, c.sunk, c.attack, c.stepCostSummary)
			}
			hits := ticks.hitsBy(int(a.InternalID))
			first, staminaSpent := 1, meeting.sunk
			if c.attack == 0 {
				first = 2
			}
			if len(hits) == 0 || hits[0].tick != first {
				t.Fatalf("A's hits %+v, want the first in tick %d", hits, first)
			}
			if want := map[bool]int{true: c.attack, false: 3}[c.attack > 0]; hits[0].stamina != want {
				t.Errorf("first hit used %d stamina, want %d", hits[0].stamina, want)
			}
			for _, h := range hits {
				staminaSpent += h.stamina
			}
			if staminaSpent != a.CurrentStamina {
				t.Errorf("A spent %d stamina on the half move and hits, want all %d it started with", staminaSpent, a.CurrentStamina)
			}
		})
	}
}
