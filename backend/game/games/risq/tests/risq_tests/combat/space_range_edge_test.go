package combat

import (
	"fmt"
	"testing"

	"github.com/dgray001/gray_online/game/games/risq/internal/defs"
	"github.com/dgray001/gray_online/game/games/risq/tests/harness"
)

var neighborDirections = []harness.Coord{{X: 1, Y: 0}, {X: 1, Y: -1}, {X: 0, Y: -1}, {X: -1, Y: 0}, {X: -1, Y: 1}, {X: 0, Y: 1}}

func opposite(c harness.Coord) harness.Coord {
	return harness.Coord{X: -c.X, Y: -c.Y}
}

// The attacker in zone attackerZone of space (0,0), a passive target in targetZone of the neighbor at dir
func edgeGame(t *testing.T, dir, attackerZone, targetZone harness.Coord) *harness.Game {
	t.Helper()
	g := spaceRangeGameTweaked(t, defs.RisqRange_SPACE, fullVision,
		spaceOf(0, 0, flat, placed{blunt, 0, attackerZone.X, attackerZone.Y}),
		spaceOf(dir.X, dir.Y, flat, placed{heavy, 1, targetZone.X, targetZone.Y}))
	makePassive(g, 1, soleUnit(g, 1))
	return g
}

// Isolates range from vision: every zone of the attacker sees its neighbors well
func fullVision() {
	config := defs.UnitConfigs[blunt]
	config.Vision = defs.RisqVision{Space: defs.VisibilityGood, Edge_adjacent: defs.VisibilityGood, Adjacent: defs.VisibilityGood, Edge_opposite: defs.VisibilityGood, Secondary: defs.VisibilityGood}
	defs.UnitConfigs[blunt] = config
}

func orderTypes() map[string]defs.OrderType {
	return map[string]defs.OrderType{"unit": defs.OrderType_UnitAttackUnit, "zone": defs.OrderType_UnitAttackZone, "space": defs.OrderType_UnitAttackSpace}
}

func TestSpaceRangeEdgeZoneShootsFacingZone(t *testing.T) {
	for _, dir := range neighborDirections {
		for name, kind := range orderTypes() {
			t.Run(fmt.Sprintf("%v-%s", dir, name), func(t *testing.T) {
				log := captureTicks(t)
				g := edgeGame(t, dir, dir, opposite(dir))
				a, b := soleUnit(g, 0), soleUnit(g, 1)
				g.Submit(g.Human(0), spaceRangeAttackOrder(a, b, kind))
				g.Submit(g.Human(1))
				ticks := parseTicks(t, log)
				hits := ticks.hitsBy(int(a.InternalID))
				if len(ticks.moves) != 0 || len(ticks.meetings) != 0 || len(hits) == 0 || hits[0].tick != 1 || hits[0].stamina != 3 {
					t.Fatalf("edge unit did not shoot the facing zone on tick 1 without moving: %+v", ticks)
				}
				if after := soleUnit(g, 0); after.Space != a.Space || after.Zone != a.Zone {
					t.Fatalf("edge unit moved: %+v", after)
				}
			})
		}
	}
}

func TestSpaceRangeEdgeZoneMustMoveForOtherZones(t *testing.T) {
	dir := harness.Coord{X: 1, Y: 0}
	for _, target := range []harness.Coord{{X: 0, Y: 0}, {X: 1, Y: 0}, {X: 0, Y: -1}, {X: -1, Y: 1}} {
		for name, kind := range orderTypes() {
			t.Run(fmt.Sprintf("%v-%s", target, name), func(t *testing.T) {
				log := captureTicks(t)
				g := edgeGame(t, dir, dir, target)
				a, b := soleUnit(g, 0), soleUnit(g, 1)
				g.Submit(g.Human(0), spaceRangeAttackOrder(a, b, kind))
				g.Submit(g.Human(1))
				ticks := parseTicks(t, log)
				hits := ticks.hitsBy(int(a.InternalID))
				if len(ticks.movesOf(int(a.InternalID))) == 0 || len(hits) == 0 || hits[0].tick == 1 {
					t.Fatalf("non-facing zone must need a move before the first shot: %+v", ticks)
				}
			})
		}
	}
}

func TestSpaceRangeCenterZoneStepsToEdgeInsteadOfCrossing(t *testing.T) {
	for name, kind := range orderTypes() {
		t.Run(name, func(t *testing.T) {
			log := captureTicks(t)
			dir := harness.Coord{X: 1, Y: 0}
			g := edgeGame(t, dir, harness.Coord{}, opposite(dir))
			a, b := soleUnit(g, 0), soleUnit(g, 1)
			g.Submit(g.Human(0), spaceRangeAttackOrder(a, b, kind))
			g.Submit(g.Human(1))
			ticks := parseTicks(t, log)
			after := soleUnit(g, 0)
			if after.Space != a.Space || after.Zone != dir || len(ticks.movesOf(int(a.InternalID))) != 1 || len(ticks.hitsBy(int(a.InternalID))) == 0 {
				t.Fatalf("want one step to the facing edge inside the space and then shots: %+v unit=%+v", ticks, after)
			}
		})
	}
}
