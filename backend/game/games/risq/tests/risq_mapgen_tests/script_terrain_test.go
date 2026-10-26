package risq_mapgen_tests

import (
	"strconv"
	"strings"
	"testing"

	"github.com/dgray001/gray_online/game/game_utils"
	"github.com/dgray001/gray_online/game/games/risq/internal/defs"
	"github.com/dgray001/gray_online/game/games/risq/tests/harness/fakeboard"
)

const (
	hexagon3    = `{"step":"shape","params":{"kind":"hexagon","size":{"radius":3}}}`
	ring3       = `{"step":"shape","params":{"kind":"ring","size":{"outer_radius":3,"inner_radius":1}}}`
	flatTerrain = `{"step":"terrain_fill","params":{"terrain_id":1}}`
)

// Coordinates of every space with the given terrain
func withTerrain(board *fakeboard.Board, terrain uint32) []game_utils.Coordinate2D {
	var coords []game_utils.Coordinate2D
	for _, info := range board.Inspect() {
		if info.Terrain == terrain {
			coords = append(coords, info.Coord)
		}
	}
	return coords
}

const hexagon4 = `{"step":"shape","params":{"kind":"hexagon","size":{"radius":4}}}`

func blobScript(shape string, params string) string {
	return `[` + shape + `,` + flatTerrain + `,` + step("terrain_blob", params) + `]`
}

func TestRingPlacementKeepInsideFillsExactlyTheRing(t *testing.T) {
	ring := `"terrain_id":21,"seed_count":1,"size":200,"min_spacing":0,"placement":{"pattern":"ring","distance":[3,3],"keep_inside":true}`
	useScripts(t, map[string]string{"ring": blobScript(hexagon4, `{`+ring+`}`)})
	painted := withTerrain(generateStartless(t, "ring", 2), 21)
	if len(painted) != 18 {
		t.Fatalf("painted %d spaces, want the 18 on distance 3", len(painted))
	}
	for _, c := range painted {
		if d := game_utils.AxialDistance(game_utils.Coordinate2D{}, c); d != 3 {
			t.Errorf("painted %v at distance %d", c, d)
		}
	}
}

func TestRingPlacementWithoutKeepInsideLeaksOutOfTheRing(t *testing.T) {
	ring := `"terrain_id":21,"seed_count":1,"size":30,"min_spacing":0,"placement":{"pattern":"ring","distance":[3,3]}`
	useScripts(t, map[string]string{"leak": blobScript(hexagon4, `{`+ring+`}`)})
	for _, c := range withTerrain(generateStartless(t, "leak", 2), 21) {
		if game_utils.AxialDistance(game_utils.Coordinate2D{}, c) != 3 {
			return
		}
	}
	t.Error("a 30 space blob stayed on a ring of 18 spaces without keep_inside")
}

func TestRingBandKeepInsideReplacesTheBorderStep(t *testing.T) {
	ring := `"terrain_id":21,"seed_count":1,"size":500,"min_spacing":0,"placement":{"pattern":"ring","distance":[3,4],"keep_inside":true}`
	useScripts(t, map[string]string{"band": blobScript(hexagon4, `{`+ring+`}`)})
	if got := len(withTerrain(generateStartless(t, "band", 2), 21)); got != 42 {
		t.Errorf("painted %d spaces, want the 18 + 24 of distances 3 and 4", got)
	}
}

func TestPointsPlacementSeedsExactlyTheListedSpaces(t *testing.T) {
	points := `"terrain_id":21,"seed_count":99,"size":1,"min_spacing":0,"placement":{"pattern":"points","points":[{"x":0,"y":0},{"x":2,"y":-1}]}`
	useScripts(t, map[string]string{"points": blobScript(hexagon4, `{`+points+`}`)})
	if got := len(withTerrain(generateStartless(t, "points", 2), 21)); got != 2 {
		t.Errorf("painted %d spaces, want the 2 listed points", got)
	}
}

func TestHardAndPreferredSpacingModes(t *testing.T) {
	blob := func(spacing string) string {
		return blobScript(hexagon3, `{"terrain_id":21,"seed_count":10,"size":1,`+spacing+`}`)
	}
	useScripts(t, map[string]string{"hard": blob(`"min_spacing":5`), "soft": blob(`"preferred_spacing":5`), "both": blob(`"min_spacing":3,"preferred_spacing":5`)})
	hard := withTerrain(generateStartless(t, "hard", 2), 21)
	soft := withTerrain(generateStartless(t, "soft", 2), 21)
	both := withTerrain(generateStartless(t, "both", 2), 21)
	if len(hard) >= 10 || len(soft) != 10 || len(both) <= len(hard) || len(both) >= 10 {
		t.Errorf("hard %d, soft %d, both %d seeds, want hard < both < 10 and soft == 10", len(hard), len(soft), len(both))
	}
	for i, a := range both {
		for _, b := range both[i+1:] {
			if game_utils.AxialDistance(a, b) < 3 {
				t.Errorf("seeds %v and %v are closer than the hard floor of 3", a, b)
			}
		}
	}
}

func TestKeepInsideRequiresTheRingPattern(t *testing.T) {
	bad := `"terrain_id":21,"seed_count":1,"size":5,"min_spacing":0,"placement":{"pattern":"random","keep_inside":true}`
	useScripts(t, map[string]string{"bad": blobScript(hexagon3, `{`+bad+`}`)})
	if _, err := fakeboard.Generate("script:bad", 2, 1); err == nil || !strings.Contains(err.Error(), "keep_inside") {
		t.Errorf("got %v, want a keep_inside error", err)
	}
}

const patchBlob = `{"step":"terrain_blob","params":{"terrain_id":21,"seed_count":1,"size":7,"min_spacing":0,"placement":{"pattern":"ring","distance":[0,0]}}}`

func TestTerrainFillOnlyPaintsMatchingTerrain(t *testing.T) {
	repaint := step("terrain_fill", `{"terrain_id":9,"on":{"terrain_ids":[1]}}`)
	useScripts(t, map[string]string{"filtered": `[` + hexagon3 + `,` + flatTerrain + `,` + patchBlob + `,` + repaint + `]`})
	board := generateStartless(t, "filtered", 2)
	if patch, painted := len(withTerrain(board, 21)), len(withTerrain(board, 9)); patch != 7 || painted != 30 {
		t.Errorf("%d patch and %d repaint spaces, want the 7 patch kept and the other 30 painted", patch, painted)
	}
}

func TestTerrainBlobOnlySeedsAndGrowsOverMatchingTerrain(t *testing.T) {
	repaint := step("terrain_blob", `{"terrain_id":9,"seed_count":1,"size":200,"min_spacing":0,"on":{"terrain_names":["Flat H"]}}`)
	useScripts(t, map[string]string{"filtered": `[` + hexagon3 + `,` + flatTerrain + `,` + patchBlob + `,` + repaint + `]`})
	board := generateStartless(t, "filtered", 2)
	if repaint, patch, flatTerrain := len(withTerrain(board, 9)), len(withTerrain(board, 21)), len(withTerrain(board, 1)); repaint != 7 || patch != 0 || flatTerrain != 30 {
		t.Errorf("%d repaint, %d patch, %d flat, want the whole patch (7) repainted and nothing else", repaint, patch, flatTerrain)
	}
}

func TestRingFromEdgeFollowsTheOutlineAndFromCenterTheShapeCenter(t *testing.T) {
	ring := func(shape string, from string, lo int, hi int) string {
		return blobScript(shape, `{"terrain_id":21,"seed_count":1,"size":500,"min_spacing":0,"placement":{"pattern":"ring","from":"`+from+`","distance":[`+strconv.Itoa(lo)+`,`+strconv.Itoa(hi)+`],"keep_inside":true}}`)
	}
	triangle := `{"step":"shape","params":{"kind":"triangle","size":{"edge_length":6}}}`
	useScripts(t, map[string]string{"outline": ring(hexagon4, "edge", 0, 0), "inset": ring(hexagon4, "edge", 1, 1), "middle": ring(triangle, "center", 0, 0)})
	for name, want := range map[string]int{"outline": 24, "inset": 18, "middle": 1} {
		if got := len(withTerrain(generateStartless(t, name, 2), 21)); got != want {
			t.Errorf("%s painted %d spaces, want %d", name, got, want)
		}
	}
}

func TestBlobSizeRangeDrawsPerBlob(t *testing.T) {
	useScripts(t, map[string]string{"ranged": blobScript(hexagon4, `{"terrain_id":21,"seed_count":1,"size":[2,4],"placement":{"pattern":"ring","distance":[0,0]}}`)})
	seen := map[int]bool{}
	for seed := int64(1); seed <= 40; seed++ {
		board, _ := fakeboard.Generate("script:ranged", 2, seed)
		seen[len(withTerrain(board, 21))] = true
	}
	if len(seen) != 3 || !seen[2] || !seen[3] || !seen[4] {
		t.Errorf("painted sizes %v, want exactly 2, 3 and 4 across seeds", seen)
	}
}

func TestTerrainFillPaintsEverySpace(t *testing.T) {
	useScripts(t, map[string]string{"fill": `[` + hexagon3 + `,` + flatTerrain + `]`})
	if got := len(withTerrain(generateStartless(t, "fill", 2), 1)); got != 37 {
		t.Errorf("%d flat spaces, want all 37", got)
	}
}

func TestVariantBlobConvertsOnlyToTheHillyVariant(t *testing.T) {
	blob := `{"variant":"hilly","seed_count":1,"size":3,"min_spacing":0,"placement":{"pattern":"ring","distance":[0,0]}}`
	useScripts(t, map[string]string{"hilly": blobScript(hexagon3, blob)})
	board := generateStartless(t, "hilly", 2)
	if hillyTerrain, flatTerrain := len(withTerrain(board, 51)), len(withTerrain(board, 1)); hillyTerrain != 3 || flatTerrain != 34 {
		t.Errorf("%d hilly and %d flat spaces, want 3 and 34", hillyTerrain, flatTerrain)
	}
}

func TestMountainousVariantOnlyGrowsOverHilly(t *testing.T) {
	hillyTerrain := `{"terrain_id":51,"seed_count":1,"size":7,"min_spacing":0,"placement":{"pattern":"ring","distance":[0,0]}}`
	mountains := step("terrain_blob", `{"variant":"mountainous","seed_count":1,"size":99,"min_spacing":0,"placement":{"pattern":"ring","distance":[0,0]},"on":{"terrain_types":["hilly"]}}`)
	useScripts(t, map[string]string{"peaks": `[` + hexagon3 + `,` + flatTerrain + `,` + step("terrain_blob", hillyTerrain) + `,` + mountains + `]`})
	board := generateStartless(t, "peaks", 2)
	if peaks, hillyTerrain, flatTerrain := len(withTerrain(board, 101)), len(withTerrain(board, 51)), len(withTerrain(board, 1)); peaks != 7 || hillyTerrain != 0 || flatTerrain != 30 {
		t.Errorf("%d mountains, %d hilly, %d flat, want the 7 hilly spaces converted and the rest untouched", peaks, hillyTerrain, flatTerrain)
	}
}

func TestVariantCannotCombineWithPaintedTerrain(t *testing.T) {
	bad := step("terrain_blob", `{"variant":"hilly","terrain_id":21,"seed_count":1,"size":3}`)
	useScripts(t, map[string]string{"bad": `[` + hexagon3 + `,` + flatTerrain + `,` + bad + `]`})
	if _, err := fakeboard.Generate("script:bad", 2, 1); err == nil || !strings.Contains(err.Error(), "variant can't be combined") {
		t.Errorf("got %v, want a variant exclusivity error", err)
	}
}

func TestTerrainFilterMatchesWholeFamilyAndAcceptsExpressions(t *testing.T) {
	hillyTerrain := step("terrain_blob", `{"terrain_id":51,"seed_count":1,"size":3,"min_spacing":0,"placement":{"pattern":"ring","distance":[0,0]}}`)
	fill := step("terrain_fill", `{"terrain_id":21,"on":{"terrain_ids":["flat_id"]}}`)
	useScripts(t, map[string]string{"family": `[` + hexagon3 + `,` + step("define", `{"name":"flat_id","value":1}`) + `,` + flatTerrain + `,` + hillyTerrain + `,` + fill + `]`})
	if got := len(withTerrain(generateStartless(t, "family", 2), 21)); got != 37 {
		t.Errorf("%d patch spaces, want all 37 because a flat id also matches its hilly variant", got)
	}
}

func TestTerrainBlobSeedRegionLine(t *testing.T) {
	line := step("terrain_blob", `{"terrain_id":21,"seed_region":"line","placement":{"points":[{"x":-2,"y":0},{"x":2,"y":0}]}}`)
	useScripts(t, map[string]string{"line": `[` + hexagon3 + `,` + flatTerrain + `,` + line + `]`})
	painted := withTerrain(generateStartless(t, "line", 2), 21)
	if len(painted) != 5 {
		t.Fatalf("got %d painted spaces, want 5 for straight line from -2 to 2", len(painted))
	}
}

func TestTerrainBlobSeedRegionRectangle(t *testing.T) {
	rect := step("terrain_blob", `{"terrain_id":21,"seed_region":"rectangle","placement":{"points":[{"x":0,"y":-1},{"x":1,"y":1}]}}`)
	useScripts(t, map[string]string{"rect": `[` + hexagon3 + `,` + flatTerrain + `,` + rect + `]`})
	painted := withTerrain(generateStartless(t, "rect", 2), 21)
	if len(painted) != 9 {
		t.Fatalf("got %d painted spaces, want 9 for 3x3 rectangle", len(painted))
	}
}

func TestTerrainBlobSeedRegionLineExpandsWithSize(t *testing.T) {
	line := step("terrain_blob", `{"terrain_id":21,"seed_region":"line","placement":{"points":[{"x":-1,"y":0},{"x":1,"y":0}]},"size":7,"roundness":1.0}`)
	useScripts(t, map[string]string{"exp": `[` + hexagon3 + `,` + flatTerrain + `,` + line + `]`})
	painted := withTerrain(generateStartless(t, "exp", 2), 21)
	if len(painted) != 7 {
		t.Fatalf("got %d painted spaces, want 7 for line of 3 grown to 7", len(painted))
	}
}

func TestRotateTerrainSixFold(t *testing.T) {
	line := step("terrain_blob", `{"terrain_id":21,"seed_region":"line","placement":{"points":[{"x":1,"y":0},{"x":3,"y":0}]}}`)
	rot := step("rotate", `{"count":6,"on":{"terrain_ids":[21]}}`)
	useScripts(t, map[string]string{"rot": `[` + hexagon3 + `,` + flatTerrain + `,` + line + `,` + rot + `]`})
	painted := withTerrain(generateStartless(t, "rot", 2), 21)
	if len(painted) != 18 {
		t.Fatalf("got %d painted spaces, want 18 for 3-tile line rotated 6 times", len(painted))
	}
}

func TestRotateTerrainRegion(t *testing.T) {
	line := step("terrain_blob", `{"terrain_id":21,"seed_region":"line","placement":{"points":[{"x":1,"y":0},{"x":3,"y":0}]},"region":"leaf"}`)
	rot := step("rotate", `{"count":6,"region":"leaf"}`)
	useScripts(t, map[string]string{"rot_reg": `[` + hexagon3 + `,` + flatTerrain + `,` + line + `,` + rot + `]`})
	painted := withTerrain(generateStartless(t, "rot_reg", 2), 21)
	if len(painted) != 18 {
		t.Fatalf("got %d painted spaces, want 18 rotated from leaf region", len(painted))
	}
}

func TestRotateCopiesSpaceGold(t *testing.T) {
	line := step("terrain_blob", `{"terrain_id":21,"seed_region":"line","space_gold":9,"placement":{"points":[{"x":1,"y":0},{"x":3,"y":0}]}}`)
	rot := step("rotate", `{"count":6,"on":{"terrain_ids":[21]}}`)
	useScripts(t, map[string]string{"rot_gold": `[` + hexagon3 + `,` + flatTerrain + `,` + line + `,` + rot + `]`})
	board := generateStartless(t, "rot_gold", 2)
	painted := map[game_utils.Coordinate2D]bool{}
	for _, c := range withTerrain(board, 21) {
		painted[c] = true
	}
	for _, space := range board.Spaces() {
		want := 0.0
		if painted[space.Coordinate()] {
			want = 9
		}
		if got := space.GoldIncome(); got != want {
			t.Errorf("space %v pays %v, want %v", space.Coordinate(), got, want)
		}
	}
}

func TestDistanceToFilter(t *testing.T) {
	center := step("terrain_blob", `{"terrain_id":51,"seed_count":1,"size":1,"placement":{"pattern":"points","points":[{"x":0,"y":0}]}}`)
	ring1 := step("terrain_fill", `{"terrain_id":21,"on":{"distance_to":{"target":{"terrain_types":["hilly"]},"distance":1}}}`)
	useScripts(t, map[string]string{"dist": `[` + hexagon3 + `,` + flatTerrain + `,` + center + `,` + ring1 + `]`})
	painted := withTerrain(generateStartless(t, "dist", 2), 21)
	if len(painted) != 6 {
		t.Fatalf("got %d painted spaces, want 6 at distance 1", len(painted))
	}
}

func TestDistanceToMinBuffer(t *testing.T) {
	center := step("terrain_blob", `{"terrain_id":51,"seed_count":1,"size":1,"placement":{"pattern":"points","points":[{"x":0,"y":0}]}}`)
	deep := step("terrain_fill", `{"terrain_id":21,"on":{"distance_to":{"target":{"terrain_types":["hilly"]},"min":2}}}`)
	useScripts(t, map[string]string{"buf": `[` + hexagon3 + `,` + flatTerrain + `,` + center + `,` + deep + `]`})
	painted := withTerrain(generateStartless(t, "buf", 2), 21)
	if len(painted) != 30 {
		t.Fatalf("got %d painted spaces, want 30 at distance >= 2", len(painted))
	}
}

func TestDistanceToCloverShoreline(t *testing.T) {
	deepWater := step("terrain_fill", `{"terrain_id":301}`)
	center := step("terrain_blob", `{"terrain_id":1,"seed_count":1,"size":1,"placement":{"pattern":"points","points":[{"x":0,"y":0}]}}`)
	shore := step("terrain_fill", `{"terrain_id":251,"on":{"terrain_types":["deep_water"],"distance_to":{"target":{"terrain_types":["flatlands"]},"distance":1}}}`)
	useScripts(t, map[string]string{"clover": `[` + hexagon3 + `,` + deepWater + `,` + center + `,` + shore + `]`})
	board := generateStartless(t, "clover", 2)
	if len(withTerrain(board, 251)) != 6 || len(withTerrain(board, 301)) != 30 {
		t.Fatalf("shore=%d want 6, deep=%d want 30", len(withTerrain(board, 251)), len(withTerrain(board, 301)))
	}
}

func TestIncludeStep(t *testing.T) {
	helper := `[` + step("terrain_fill", `{"terrain_id":21}`) + `]`
	main := `[` + hexagon3 + `,` + flatTerrain + `,` + step("include", `{"path":"sub/helper.json"}`) + `]`
	useScripts(t, map[string]string{"sub/helper": helper, "main": main})
	board := generateStartless(t, "main", 2)
	if len(withTerrain(board, 21)) != 37 {
		t.Fatalf("got %d painted spaces from include, want 37", len(withTerrain(board, 21)))
	}
}

func TestAsteriskMapScript(t *testing.T) {
	if err := defs.LoadConfig("../../config"); err != nil {
		t.Fatal(err)
	}
	defer func() { _ = defs.LoadConfig(testConfig) }()
	board, err := fakeboard.Generate("script:asterisk", 2, 1)
	if err != nil {
		t.Fatal(err)
	}
	counts := make(map[uint32]int)
	for _, s := range board.Inspect() {
		counts[s.Terrain]++
	}
	t.Logf("Asterisk spaces=%d terrains: %+v", len(board.Inspect()), counts)
}

func TestPathDijkstraWindsAroundObstacle(t *testing.T) {
	wall := step("terrain_blob", `{"terrain_id":301,"seed_region":"line","placement":{"points":[{"x":-1,"y":0},{"x":1,"y":0}]}}`)
	path := step("terrain_blob", `{"terrain_id":21,"seed_region":"path","placement":{"points":[{"x":0,"y":-1},{"x":0,"y":1}]},"path_costs":{"flatlands":1,"deep_water":100}}`)
	useScripts(t, map[string]string{"detour": `[` + hexagon3 + `,` + flatTerrain + `,` + wall + `,` + path + `]`})
	board := generateStartless(t, "detour", 2)
	painted := withTerrain(board, 21)
	if len(painted) == 0 {
		t.Fatal("expected detour path to paint spaces")
	}
	for _, c := range painted {
		if c.Y == 0 && (c.X >= -1 && c.X <= 1) {
			t.Fatalf("path stepped on wall at %v", c)
		}
	}
}

func TestPathBetweenPlayers(t *testing.T) {
	starts := step("player_starts", `{"pattern":"ring","area_size":1,"starting_distance":2,"terrain_id":1}`)
	road := step("terrain_blob", `{"terrain_id":21,"seed_region":"path","between":["player:1","player:2"]}`)
	useScripts(t, map[string]string{"road": `[` + hexagon3 + `,` + flatTerrain + `,` + starts + `,` + road + `]`})
	board, err := fakeboard.Generate("script:road", 2, 1)
	if err != nil {
		t.Fatal(err)
	}
	if len(withTerrain(board, 21)) == 0 {
		t.Fatal("expected road between player 1 and 2")
	}
}

func TestPathBetweenPlayersBeforeStartsErrors(t *testing.T) {
	road := step("terrain_blob", `{"terrain_id":21,"seed_region":"path","between":{"players":{"nearest":1}}}`)
	useScripts(t, map[string]string{"err_road": `[` + hexagon3 + `,` + flatTerrain + `,` + road + `]`})
	_, err := fakeboard.Generate("script:err_road", 2, 1)
	if err == nil || !strings.Contains(err.Error(), "requires an earlier player_starts step") {
		t.Fatalf("got %v, want error requiring earlier player_starts step", err)
	}
}

func TestPathBetweenAllPlayersNearest(t *testing.T) {
	starts := step("player_starts", `{"pattern":"ring","area_size":1,"starting_distance":2,"terrain_id":1}`)
	roads := step("terrain_blob", `{"terrain_id":21,"seed_region":"path","between":{"players":{"nearest":2}}}`)
	useScripts(t, map[string]string{"mesh": `[` + hexagon3 + `,` + flatTerrain + `,` + starts + `,` + roads + `]`})
	board, err := fakeboard.Generate("script:mesh", 3, 1)
	if err != nil {
		t.Fatal(err)
	}
	if len(withTerrain(board, 21)) < 6 {
		t.Fatalf("expected ring of roads, got %d painted spaces", len(withTerrain(board, 21)))
	}
}

func TestFilterSeeds(t *testing.T) {
	wall := step("terrain_blob", `{"terrain_id":301,"seed_region":"line","placement":{"points":[{"x":0,"y":-1},{"x":0,"y":0}]}}`)
	line := step("terrain_blob", `{"terrain_id":21,"seed_region":"line","filter_seeds":true,"on":{"terrain_types":["flatlands"]},"placement":{"points":[{"x":0,"y":-2},{"x":0,"y":1}]}}`)
	useScripts(t, map[string]string{"spotty": `[` + hexagon3 + `,` + flatTerrain + `,` + wall + `,` + line + `]`})
	board := generateStartless(t, "spotty", 2)
	painted := withTerrain(board, 21)
	for _, c := range painted {
		if c.X == 0 && (c.Y == -1 || c.Y == 0) {
			t.Fatalf("filter_seeds failed: painted on deep water wall at %v", c)
		}
	}
}

func TestTerrainStepsSetSpaceGold(t *testing.T) {
	blob := step("terrain_blob", `{"terrain_id":21,"seed_count":1,"size":3,"min_spacing":0,"space_gold":"2 * 4","placement":{"pattern":"points","points":[{"x":0,"y":0}]}}`)
	fill := step("terrain_fill", `{"terrain_id":1,"space_gold":3}`)
	useScripts(t, map[string]string{"gold": `[` + hexagon3 + `,` + fill + `,` + blob + `]`, "plain": `[` + hexagon3 + `,` + flatTerrain + `]`})
	board := generateStartless(t, "gold", 2)
	painted := map[game_utils.Coordinate2D]bool{}
	for _, c := range withTerrain(board, 21) {
		painted[c] = true
	}
	if len(painted) != 3 {
		t.Fatalf("blob painted %d spaces, want 3", len(painted))
	}
	for _, space := range board.Spaces() {
		want := 3.0
		if painted[space.Coordinate()] {
			want = 8
		}
		if got := space.(interface{ GoldIncome() float64 }).GoldIncome(); got != want {
			t.Errorf("space %v pays %v, want %v", space.Coordinate(), got, want)
		}
	}
	for _, space := range generateStartless(t, "plain", 2).Spaces() {
		if got := space.(interface{ GoldIncome() float64 }).GoldIncome(); got != 0 {
			t.Errorf("without space_gold, space %v pays %v, want the default left alone", space.Coordinate(), got)
		}
	}
}
