package combat

import (
	"slices"
	"testing"

	"github.com/dgray001/gray_online/game/games/risq/internal/defs"
	"github.com/dgray001/gray_online/game/games/risq/tests/harness"
	"github.com/gin-gonic/gin"
)

var (
	center = harness.Coord{}
	east   = harness.Coord{X: 1, Y: 0}
	west   = harness.Coord{X: -1, Y: 0}
	// placements around an attacker in space (0,0): the zone facing the attacker's east edge, and one that does not
	eastFacing = enemyPlacement{space: east, zone: west}
	eastCenter = enemyPlacement{space: east, zone: center}
	// reachable only from the west edge of space (0,0)
	westFacing = enemyPlacement{space: west, zone: east}
	sameWest   = enemyPlacement{space: center, zone: west}
)

type rangedStanceRun struct {
	hurt     []enemyPlacement
	attacker harness.Unit
	ticks    tickLog
}

// A space-range grunt in zone attackerZone of space (0,0) with the stance, against passive phalanxes at the placements, over one turn
func runRangedStance(t *testing.T, stance defs.UnitStance, attackerZone harness.Coord, enemies ...enemyPlacement) rangedStanceRun {
	t.Helper()
	log := captureTicks(t)
	spaceOrder := []harness.Coord{center}
	placedBySpace := map[harness.Coord][]placed{center: {{blunt, 0, attackerZone.X, attackerZone.Y}}}
	for _, e := range enemies {
		if _, seen := placedBySpace[e.space]; !seen {
			spaceOrder = append(spaceOrder, e.space)
		}
		placedBySpace[e.space] = append(placedBySpace[e.space], placed{heavy, 1, e.zone.X, e.zone.Y})
	}
	spaces := []string{}
	for _, space := range spaceOrder {
		spaces = append(spaces, spaceOf(space.X, space.Y, flat, placedBySpace[space]...))
	}
	g := spaceRangeGameTweaked(t, defs.RisqRange_SPACE, fullVision, spaces...)
	owner, foe := g.Human(0), g.Human(1)
	for _, unit := range g.Self(foe).Units {
		passive(g, foe, unit.InternalID)
	}
	attacker := g.Self(owner).Units[0]
	g.Action(owner, "set-unit-behavior", gin.H{"internal_ids": []uint64{attacker.InternalID}, "stance": uint8(stance), "attack_back": false})
	g.EndTurn()
	run := rangedStanceRun{attacker: g.Self(owner).Units[0], ticks: parseTicks(t, log)}
	for _, unit := range g.Self(foe).Units {
		if unit.CombatStats.Health < float64(unit.CombatStats.MaxHealth) {
			run.hurt = append(run.hurt, enemyPlacement{unit.Space, unit.Zone})
		}
	}
	return run
}

func TestRangedStanceReach(t *testing.T) {
	for _, c := range []struct {
		name     string
		stance   defs.UnitStance
		attacker harness.Coord
		enemy    enemyPlacement
		hurt     bool
		endsAt   *harness.Coord
	}{
		{"stand ground: edge hits facing zone", defs.UnitStance_STAND_GROUND, east, eastFacing, true, &east},
		{"stand ground: edge hits own space", defs.UnitStance_STAND_GROUND, east, sameWest, true, &east},
		{"stand ground: edge ignores neighbor center", defs.UnitStance_STAND_GROUND, east, eastCenter, false, &east},
		{"stand ground: center never steps to an edge", defs.UnitStance_STAND_GROUND, center, eastFacing, false, &center},
		{"defensive: edge hits facing zone", defs.UnitStance_DEFENSIVE, east, eastFacing, true, &east},
		{"defensive: center steps to the facing edge", defs.UnitStance_DEFENSIVE, center, eastFacing, true, &east},
		{"defensive: edge moves to the opposite edge", defs.UnitStance_DEFENSIVE, east, westFacing, true, &west},
		{"defensive: edge ignores neighbor center", defs.UnitStance_DEFENSIVE, east, eastCenter, false, &east},
		{"defensive: center ignores neighbor center", defs.UnitStance_DEFENSIVE, center, eastCenter, false, &center},
		{"aggressive: edge hits facing zone", defs.UnitStance_AGGRESSIVE, east, eastFacing, true, &east},
		{"aggressive: center steps to the facing edge", defs.UnitStance_AGGRESSIVE, center, eastFacing, true, &east},
		{"aggressive: edge chases into the neighbor", defs.UnitStance_AGGRESSIVE, east, eastCenter, true, nil},
	} {
		t.Run(c.name, func(t *testing.T) {
			run := runRangedStance(t, c.stance, c.attacker, c.enemy)
			if hurt := len(run.hurt) == 1; hurt != c.hurt {
				t.Errorf("hurt=%v, want %v", run.hurt, c.hurt)
			}
			if c.endsAt != nil && (run.attacker.Space != center || run.attacker.Zone != *c.endsAt) {
				t.Errorf("attacker ended at %v/%v, want %v in its own space", run.attacker.Space, run.attacker.Zone, *c.endsAt)
			}
			if !c.hurt && len(run.attacker.ActiveOrders) != 0 {
				t.Errorf("unreachable target left an order: %+v", run.attacker.ActiveOrders)
			}
		})
	}
}

// A target already in range beats one the attacker must walk to, whatever the ids
func TestRangedDefensivePrefersTargetInRange(t *testing.T) {
	for name, enemies := range map[string][]enemyPlacement{
		"in-range first": {eastFacing, westFacing},
		"in-range last":  {westFacing, eastFacing},
	} {
		t.Run(name, func(t *testing.T) {
			run := runRangedStance(t, defs.UnitStance_DEFENSIVE, east, enemies...)
			if !slices.Equal(run.hurt, []enemyPlacement{eastFacing}) || run.attacker.Zone != east {
				t.Fatalf("hurt=%v attacker zone=%v, want only the in-range target from the starting zone", run.hurt, run.attacker.Zone)
			}
		})
	}
}

// Aggressive scores every zone of an adjacent space alike unless range is considered
func TestRangedAggressivePrefersZoneInRange(t *testing.T) {
	for name, enemies := range map[string][]enemyPlacement{
		"in-range first": {eastFacing, eastCenter},
		"in-range last":  {eastCenter, eastFacing},
	} {
		t.Run(name, func(t *testing.T) {
			run := runRangedStance(t, defs.UnitStance_AGGRESSIVE, east, enemies...)
			hits := run.ticks.hitsBy(int(run.attacker.InternalID))
			if !slices.Equal(run.hurt, []enemyPlacement{eastFacing}) || len(run.ticks.moves) != 0 || len(hits) == 0 || hits[0].tick != 1 {
				t.Fatalf("hurt=%v ticks=%+v, want a tick 1 shot at the in-range zone without moving", run.hurt, run.ticks)
			}
		})
	}
}
