package world

import (
	"fmt"
	"testing"

	"github.com/dgray001/gray_online/game/games/risq/internal/defs"
	"github.com/dgray001/gray_online/game/games/risq/tests/harness"
)

func startWithGruntVision(t *testing.T, start uint8, source int, targets string) harness.State {
	t.Helper()
	t.Cleanup(func() { pristine.Restore(baseDir) })
	config := defs.UnitConfigs[grunt]
	config.Vision = defs.RisqVision{Space: start, Edge_adjacent: start, Adjacent: start, Edge_opposite: start, Secondary: start}
	defs.UnitConfigs[grunt] = config
	spaces := fmt.Sprintf(`{"x":0,"y":0,"terrain":%d,"zones":[{"x":1,"y":0,"units":[%s]}]}`, source, unit(grunt, 0)) + "," +
		spaceWith(0, 4, flatTerrain, unit(worker, 1)) + "," + targets
	g := startGame(t, mapDoc("", spaces))
	return g.State(g.Human(0))
}

func TestHeightDropsVisionIntoHigherTerrain(t *testing.T) {
	poorFloor := map[uint8][2]uint8{4: {3, 2}, 3: {2, 2}, 2: {2, 0}, 0: {0, 0}}
	goodFloor := map[uint8][2]uint8{4: {3, 3}, 3: {3, 2}, 2: {2, 0}, 0: {0, 0}}
	tiers := []struct {
		name           string
		x, y           int
		expected       map[uint8][2]uint8
		afterHillBoost map[uint8]uint8
	}{
		{"edge_adjacent", 1, 0, goodFloor, map[uint8]uint8{2: 3, 0: 2}},
		{"adjacent", 1, -1, poorFloor, map[uint8]uint8{2: 2, 0: 2}},
		{"edge_opposite", -1, 0, poorFloor, map[uint8]uint8{0: 2}},
		{"secondary", 2, -1, poorFloor, nil},
	}
	climbs := []struct {
		name                  string
		source, target, drops int
	}{
		{"flat_to_hill", flatTerrain, hillyTerrain, 1},
		{"hill_to_mountain", hillyTerrain, mountainousTerrain, 1},
		{"flat_to_mountain", flatTerrain, mountainousTerrain, 2},
	}
	for _, climb := range climbs {
		for _, tier := range tiers {
			for start, results := range tier.expected {
				zeroIsTrivial := start == 0 && climb.source == flatTerrain && !(climb.drops == 2 && tier.name == "adjacent")
				if zeroIsTrivial {
					continue
				}
				want := int(results[climb.drops-1])
				if boosted, ok := tier.afterHillBoost[start]; ok && climb.source == hillyTerrain {
					want = int(boosted)
				}
				t.Run(fmt.Sprintf("%s/%s/start_%d", climb.name, tier.name, start), func(t *testing.T) {
					targets := fmt.Sprintf(`{"x":%d,"y":%d,"terrain":%d}`, tier.x, tier.y, climb.target)
					if tier.name == "secondary" {
						targets += "," + spaceWith(1, 0, flatTerrain, "") + "," + spaceWith(1, -1, flatTerrain, "")
					}
					state := startWithGruntVision(t, start, climb.source, targets)
					if got := state.Space(tier.x, tier.y).Visibility; got != want {
						t.Errorf("%s from %d with %d drops: visibility %d, want %d", tier.name, start, climb.drops, got, want)
					}
				})
			}
		}
	}
}

func TestSecondaryVisionIsBlockedOnlyWhenEveryCandidateBlocks(t *testing.T) {
	cases := []struct {
		name   string
		spaces string
		x, y   int
		want   int
	}{
		{"single_candidate_blocks", spaceWith(1, 0, hillyTerrain, "") + "," + spaceWith(2, 0, flatTerrain, ""), 2, 0, unexplored},
		{"both_candidates_block", spaceWith(1, 0, hillyTerrain, "") + "," + spaceWith(1, -1, hillyTerrain, "") + "," + spaceWith(2, -1, hillyTerrain, ""), 2, -1, unexplored},
		{"one_candidate_open", spaceWith(1, 0, flatTerrain, "") + "," + spaceWith(1, -1, hillyTerrain, "") + "," + spaceWith(2, -1, hillyTerrain, ""), 2, -1, good},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			state := startWithGruntVision(t, 4, flatTerrain, c.spaces)
			if got := state.Space(c.x, c.y).Visibility; got != c.want {
				t.Errorf("secondary (%d,%d) visibility %d, want %d", c.x, c.y, got, c.want)
			}
		})
	}
}

func TestHillBuildingVisionAgainstTallerEqualAndLowerSpaces(t *testing.T) {
	spaces := fmt.Sprintf(`{"x":0,"y":0,"terrain":%d,"zones":[{"x":0,"y":0,"building":{"id":207,"player":0}}]}`, hillyTerrain) + "," +
		spaceWith(1, 0, mountainousTerrain, "") + "," + spaceWith(2, 0, flatTerrain, "") + "," +
		spaceWith(-1, 0, hillyTerrain, "") + "," + spaceWith(-2, 0, flatTerrain, "") + "," +
		spaceWith(0, -1, flatTerrain, "") + "," + spaceWith(0, 4, flatTerrain, unit(worker, 1))
	g := startGame(t, mapDoc("", spaces))
	state := g.State(g.Human(0))
	for _, c := range []struct{ x, y, want int }{{1, 0, poor}, {2, 0, unexplored}, {-1, 0, good}, {-2, 0, poor}, {0, -1, good}} {
		if got := state.Space(c.x, c.y).Visibility; got != c.want {
			t.Errorf("(%d,%d) visibility %d from a hill building, want %d", c.x, c.y, got, c.want)
		}
	}
}

func TestSourceInsideASpaceOutranksAnotherSourcesHeightDrop(t *testing.T) {
	spaces := watchtowerAt(0, 0) + "," + spaceWith(1, 0, hillyTerrain, unit(grunt, 0)) + "," + spaceWith(0, 4, flatTerrain, unit(worker, 1))
	g := startGame(t, mapDoc("", spaces))
	if got := g.State(g.Human(0)).Space(1, 0).Visibility; got != good {
		t.Errorf("(1,0) visibility %d, want %d from the unit standing in it", got, good)
	}
}

func watchtowerAt(x, y int) string {
	return fmt.Sprintf(`{"x":%d,"y":%d,"terrain":1,"zones":[{"x":0,"y":0,"building":{"id":207,"player":0}}]}`, x, y)
}

func TestSpaceStaysVisibleWhileAnotherSourceCoversIt(t *testing.T) {
	spaces := spaceWith(0, 0, flatTerrain, unit(worker, 0)) + "," + spaceWith(1, 0, flatTerrain, "") + "," + watchtowerAt(2, 0) + "," +
		spaceWith(-1, 0, flatTerrain, "") + "," + spaceWith(-2, 0, flatTerrain, "") + "," + spaceWith(3, 0, flatTerrain, unit(worker, 1))
	g := startGame(t, mapDoc("", spaces))
	west := g.Human(0)
	if moveUntilArrived(g, west, -2, 0, 6) < 1 {
		t.Fatal("the worker never walked west")
	}
	state := g.State(west)
	if got := state.Space(1, 0).Visibility; got != good {
		t.Errorf("(1,0) visibility %d after the worker left, want good from the watchtower", got)
	}
	if got := state.Space(0, 0).Visibility; got != poor {
		t.Errorf("(0,0) visibility %d, want poor from the watchtower's second ring, not fog", got)
	}
}

func TestLosingTheLastUnitInASpaceKeepsPoorVisionForOneTurn(t *testing.T) {
	spaces := spaceWith(0, 0, flatTerrain, unit(worker, 0)+","+unit(grunt, 1)) + "," + spaceWith(0, 4, flatTerrain, unit(worker, 0))
	g := startGame(t, mapDoc("", spaces))
	west := g.Human(0)
	for turn := 0; turn < 5 && len(g.Self(west).Units) > 1; turn++ {
		g.EndTurn()
	}
	if units := len(g.Self(west).Units); units != 1 {
		t.Fatalf("setup: P0 has %d units, want the worker at (0,0) killed", units)
	}
	if got := g.State(west).Space(0, 0).Visibility; got != poor {
		t.Errorf("(0,0) visibility %d the turn its last unit died, want poor", got)
	}
	g.EndTurn()
	if got := g.State(west).Space(0, 0).Visibility; got != fog {
		t.Errorf("(0,0) visibility %d a turn later, want death vision expired to fog", got)
	}
}

// An watchtower at (0,0) with no unit near it: P0's worker is far away at (0,4) and P1's at (3,0)
func TestBuildingsAloneProvideTheirConfiguredVision(t *testing.T) {
	spaces := watchtowerAt(0, 0) + "," + spaceWith(1, 0, flatTerrain, "") + "," + spaceWith(2, 0, flatTerrain, "") + "," +
		spaceWith(3, 0, flatTerrain, unit(worker, 1)) + "," + spaceWith(0, 4, flatTerrain, unit(worker, 0))
	g := startGame(t, mapDoc("", spaces))
	state := g.State(g.Human(0))
	for x, want := range []int{good, good, poor, unexplored} {
		if got := state.Space(x, 0).Visibility; got != want {
			t.Errorf("space (%d,0) visibility %d, want %d from the watchtower's rings", x, got, want)
		}
	}
}

func TestHillyTerrainCustomUnitTierAdjustments(t *testing.T) {
	spaces := fmt.Sprintf(`{"x":0,"y":0,"terrain":%d,"zones":[{"x":1,"y":0,"units":[%s]}]}`, hillyTerrain, unit(longbow, 0)) + "," +
		spaceWith(1, 0, flatTerrain, "") + "," + spaceWith(1, -1, flatTerrain, "") + "," +
		spaceWith(-1, 0, flatTerrain, "") + "," + spaceWith(2, 0, flatTerrain, "") + "," +
		spaceWith(0, 4, flatTerrain, unit(worker, 1))
	g := startGame(t, mapDoc("", spaces))
	state := g.State(g.Human(0))
	if got := state.Space(0, 0).Visibility; got != good {
		t.Errorf("own space visibility %d, want %d", got, good)
	}
	if got := state.Space(1, 0).Visibility; got != good {
		t.Errorf("edge-adjacent (1 tier lower) visibility %d, want %d", got, good)
	}
	if got := state.Space(1, -1).Visibility; got != poor {
		t.Errorf("adjacent (3 tiers lower) visibility %d, want %d", got, poor)
	}
	if got := state.Space(-1, 0).Visibility; got != good {
		t.Errorf("edge-opposite (1 tier higher) visibility %d, want %d", got, good)
	}
	if got := state.Space(2, 0).Visibility; got != good {
		t.Errorf("secondary (3 tiers higher) visibility %d, want %d", got, good)
	}
}

func TestHillyTerrainDefaultUnitBoost(t *testing.T) {
	spaces := fmt.Sprintf(`{"x":0,"y":0,"terrain":%d,"zones":[{"x":1,"y":0,"units":[%s]}]}`, hillyTerrain, unit(worker, 0)) + "," +
		spaceWith(1, -1, flatTerrain, "") + "," + spaceWith(-1, 0, flatTerrain, "") + "," +
		spaceWith(0, 4, flatTerrain, unit(worker, 1))
	g := startGame(t, mapDoc("", spaces))
	state := g.State(g.Human(0))
	if got := state.Space(1, -1).Visibility; got != good {
		t.Errorf("adjacent visibility on hills %d, want %d", got, good)
	}
	if got := state.Space(-1, 0).Visibility; got != poor {
		t.Errorf("edge-opposite visibility on hills %d, want %d", got, poor)
	}
}

func TestMountainousTerrainDefaultUnitBoost(t *testing.T) {
	spaces := fmt.Sprintf(`{"x":0,"y":0,"terrain":%d,"zones":[{"x":1,"y":0,"units":[%s]}]}`, mountainousTerrain, unit(worker, 0)) + "," +
		spaceWith(1, 0, flatTerrain, "") + "," + spaceWith(1, -1, flatTerrain, "") + "," +
		spaceWith(-1, 0, flatTerrain, "") + "," + spaceWith(2, 0, flatTerrain, "") + "," +
		spaceWith(0, 4, flatTerrain, unit(worker, 1))
	g := startGame(t, mapDoc("", spaces))
	state := g.State(g.Human(0))
	if got := state.Space(1, -1).Visibility; got != good {
		t.Errorf("adjacent visibility on mountains %d, want %d", got, good)
	}
	if got := state.Space(-1, 0).Visibility; got != good {
		t.Errorf("edge-opposite visibility on mountains %d, want %d", got, good)
	}
	if got := state.Space(2, 0).Visibility; got != poor {
		t.Errorf("secondary visibility on mountains %d, want %d", got, poor)
	}
}

func TestMountainousTerrainCustomUnitTierAdjustments(t *testing.T) {
	spaces := fmt.Sprintf(`{"x":0,"y":0,"terrain":%d,"zones":[{"x":1,"y":0,"units":[%s]}]}`, mountainousTerrain, unit(longbow, 0)) + "," +
		spaceWith(1, -1, flatTerrain, "") + "," + spaceWith(1, 0, flatTerrain, "") + "," +
		spaceWith(0, 4, flatTerrain, unit(worker, 1)) + "," +
		fmt.Sprintf(`{"x":-4,"y":4,"terrain":%d,"zones":[{"x":0,"y":0,"building":{"id":207,"player":0}}]}`, mountainousTerrain) + "," +
		spaceWith(-4, 3, flatTerrain, "") + "," + spaceWith(-3, 3, flatTerrain, "") + "," + spaceWith(-3, 2, flatTerrain, "")
	g := startGame(t, mapDoc("", spaces))
	state := g.State(g.Human(0))
	if got := state.Space(1, -1).Visibility; got != poor {
		t.Errorf("adjacent (3 tiers lower) visibility on mountains %d, want %d", got, poor)
	}
	if got := state.Space(1, 0).Visibility; got != good {
		t.Errorf("edge-adjacent (1 tier lower) visibility on mountains %d, want %d", got, good)
	}
	if got := state.Space(-3, 2).Visibility; got != good {
		t.Errorf("watchtower secondary (poor on flat) visibility on mountains %d, want %d", got, good)
	}
}
