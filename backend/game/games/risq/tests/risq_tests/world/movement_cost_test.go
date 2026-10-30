package world

import (
	"slices"
	"testing"

	"github.com/dgray001/gray_online/game/games/risq/internal/defs"
	"github.com/dgray001/gray_online/game/games/risq/tests/harness"
)

const pathingTech = 307

// P0's lone worker starts in the given space; both players hold the pathing tech with the given reductions
func pathingGame(t *testing.T, inter, intra int, start string, spaces ...string) (*harness.Game, int) {
	t.Helper()
	slow := defs.UnitConfigs[worker]
	slow.Turn_stamina = 8
	defs.UnitConfigs[worker] = slow
	tech := defs.TechConfigs[pathingTech]
	tech.Reduce_inter_move_cost, tech.Reduce_intra_move_cost = inter, intra
	defs.TechConfigs[pathingTech] = tech
	all := start + "," + spaceWith(3, -3, flatTerrain, unit(worker, 1))
	for _, s := range spaces {
		all += "," + s
	}
	g := startGame(t, mapDoc(`,"starting_techs":[307]`, all))
	return g, g.Human(0)
}

func executedMoveCosts(g *harness.Game, human int) []int {
	costs := []int{}
	for _, a := range firstUnit(g, human).TickActions {
		if a.Intent.Kind == "move" && a.Execute.Outcome == "executed" && a.Execute.StaminaSpent > 0 {
			costs = append(costs, a.Execute.StaminaSpent)
		}
	}
	return costs
}

func TestInterMoveCostReductionLowersBorderCrossing(t *testing.T) {
	cases := map[string]struct {
		reduction int
		want      []int
	}{
		"none":         {0, []int{1, 6}},
		"partial":      {2, []int{1, 4}},
		"floored-at-1": {100, []int{1, 1}},
	}
	for name, c := range cases {
		t.Run(name, func(t *testing.T) {
			g, mover := pathingGame(t, c.reduction, 0, spaceWith(0, 0, flatTerrain, unit(worker, 0)), spaceWith(1, 0, flatTerrain, ""))
			if turns := moveUntilArrived(g, mover, 1, 0, 4); turns != 1 {
				t.Fatalf("arrived after %d turns, want 1", turns)
			}
			if costs := executedMoveCosts(g, mover); !slices.Equal(costs, c.want) {
				t.Errorf("move steps cost %v, want %v", costs, c.want)
			}
		})
	}
}

func TestIntraMoveCostReductionLowersZoneStep(t *testing.T) {
	cases := map[string]struct{ reduction, want int }{
		"none":         {0, 2},
		"partial":      {1, 1},
		"floored-at-1": {100, 1},
	}
	for name, c := range cases {
		t.Run(name, func(t *testing.T) {
			g, mover := pathingGame(t, 0, c.reduction, spaceWith(0, 0, hillyTerrain, unit(worker, 0)))
			g.Submit(mover, harness.OrderMoveZone([]uint64{firstUnit(g, mover).InternalID}, 0, 0, 1, 0))
			g.EndTurn()
			if costs := executedMoveCosts(g, mover); !slices.Equal(costs, []int{c.want}) {
				t.Errorf("hilly zone step cost %v, want [%d]", costs, c.want)
			}
		})
	}
}

func TestInterMoveCostReductionMakesHillyCrossingAffordable(t *testing.T) {
	cases := map[string]struct{ reduction, turns int }{
		"none":    {0, 2},
		"reduced": {3, 1},
	}
	for name, c := range cases {
		t.Run(name, func(t *testing.T) {
			g, mover := pathingGame(t, c.reduction, 0, spaceWith(0, 0, flatTerrain, unit(worker, 0)), spaceWith(1, 0, hillyTerrain, ""))
			if turns := moveUntilArrived(g, mover, 1, 0, 4); turns != c.turns {
				t.Errorf("arrived after %d turns, want %d", turns, c.turns)
			}
		})
	}
}

func TestReducedCostsStillRouteAroundImpassableSpaces(t *testing.T) {
	g, mover := pathingGame(t, 100, 100, spaceWith(0, 0, flatTerrain, unit(worker, 0)), spaceWith(1, 0, waterTerrain, ""), spaceWith(1, -1, flatTerrain, ""), spaceWith(2, -1, flatTerrain, ""), spaceWith(2, 0, flatTerrain, ""))
	g.Submit(mover, harness.OrderMove([]uint64{firstUnit(g, mover).InternalID}, 2, 0))
	for turn := 0; turn < 6 && where(g, mover) != (harness.Coord{X: 2, Y: 0}); turn++ {
		g.EndTurn()
		for _, step := range firstUnit(g, mover).MovePath {
			if step.Space == (harness.Coord{X: 1, Y: 0}) {
				t.Fatalf("the planned route %v passes through the water space", firstUnit(g, mover).MovePath)
			}
		}
	}
	if at := where(g, mover); at != (harness.Coord{X: 2, Y: 0}) {
		t.Errorf("unit at %v, want it to have walked around the water to (2,0)", at)
	}
}
