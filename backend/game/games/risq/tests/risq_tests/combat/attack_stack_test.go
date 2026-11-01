package combat

import (
	"fmt"
	"slices"
	"testing"

	"github.com/dgray001/gray_online/game/games/risq/internal/defs"
	"github.com/dgray001/gray_online/game/games/risq/tests/harness"
	"github.com/dgray001/gray_online/game/games/risq/tests/harness/fakeboard"
	"github.com/gin-gonic/gin"
)

func attackStackGame(t *testing.T, attackers, targets int) *harness.Game {
	t.Helper()
	zone := fmt.Sprintf(`{"x":1,"y":0,"units":[{"id":21,"player":0,"count":%d},{"id":22,"player":1,"count":%d}]}`, attackers, targets)
	doc := fmt.Sprintf(`{"board_size":1,"players":2,"spaces":[{"x":0,"y":0,"terrain":1,"zones":[%s]}]}`, zone)
	fakeboard.UseConfig(t, testConfig, nil, map[string]string{"stack": doc})
	return harness.NewGame(t, "custom:stack", 1, 2)
}

func firstTickTargets(g *harness.Game, attackers []harness.Unit) map[uint64]int {
	counts := make(map[uint64]int)
	for _, attacker := range attackers {
		for _, action := range attacker.TickActions {
			if action.Tick == 1 && action.Order.Source == "stance" {
				counts[uint64(action.Order.TargetID)]++
			}
		}
	}
	return counts
}

func TestAttackStackMixedBucketsIgnoreUnitIds(t *testing.T) {
	for _, zero_first := range []bool{true, false} {
		g := attackStackGame(t, 2, 2)
		p0, p1 := g.Human(0), g.Human(1)
		attackers, targets := g.Self(p0).Units, g.Self(p1).Units
		stacks := []uint8{0, 1}
		if !zero_first {
			stacks = []uint8{1, 0}
		}
		for i, attacker := range attackers {
			g.Action(p0, "set-unit-behavior", gin.H{"internal_ids": []uint64{attacker.InternalID}, "stance": uint8(defs.UnitStance_STAND_GROUND), "attack_stack": stacks[i]})
		}
		for _, target := range targets {
			g.Action(p1, "set-unit-behavior", gin.H{"internal_ids": []uint64{target.InternalID}, "stance": uint8(defs.UnitStance_PASSIVE), "attack_back": false})
		}
		g.Submit(p0)
		g.Submit(p1)
		counts := firstTickTargets(g, g.Self(p0).Units)
		if len(counts) != 2 {
			t.Errorf("zero_first=%v: attackers per target = %v, want one on each", zero_first, counts)
		}
	}
}

func TestAttackStackSplit(t *testing.T) {
	cases := []struct {
		name  string
		stack uint8
		want  []int
	}{
		{"zero means no limit", 0, []int{12, 0, 0}},
		{"fills each target then wraps", 5, []int{5, 5, 2}},
		{"equalizes once all are full", 3, []int{4, 4, 4}},
		{"one per target then wraps", 1, []int{4, 4, 4}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			g := attackStackGame(t, 12, 3)
			p0, p1 := g.Human(0), g.Human(1)
			attackers, targets := g.Self(p0).Units, g.Self(p1).Units
			ids := make([]uint64, len(attackers))
			for i, attacker := range attackers {
				ids[i] = attacker.InternalID
			}
			g.Action(p0, "set-unit-behavior", gin.H{"internal_ids": ids, "stance": uint8(defs.UnitStance_STAND_GROUND), "attack_stack": c.stack})
			for _, target := range targets {
				g.Action(p1, "set-unit-behavior", gin.H{"internal_ids": []uint64{target.InternalID}, "stance": uint8(defs.UnitStance_PASSIVE), "attack_back": false})
			}
			g.Submit(p0)
			g.Submit(p1)
			target_ids := make([]uint64, len(targets))
			for i, target := range targets {
				target_ids[i] = target.InternalID
			}
			slices.Sort(target_ids)
			counts := firstTickTargets(g, g.Self(p0).Units)
			got := make([]int, len(target_ids))
			for i, id := range target_ids {
				got[i] = counts[id]
			}
			if !slices.Equal(got, c.want) {
				t.Errorf("attackers per target = %v, want %v", got, c.want)
			}
		})
	}
}
