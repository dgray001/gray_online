package risq_mapgen_tests

import (
	"strconv"
	"strings"
	"testing"

	"github.com/dgray001/gray_online/game/game_utils"
	"github.com/dgray001/gray_online/game/games/risq/tests/harness/fakeboard"
)

const (
	hexagon3 = `{"step":"shape","params":{"kind":"hexagon","size":{"radius":3}}}`
	ring3    = `{"step":"shape","params":{"kind":"ring","size":{"outer_radius":3,"inner_radius":1}}}`
	grass    = `{"step":"terrain_fill","params":{"terrain_id":1}}`
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
	return `[` + shape + `,` + grass + `,` + step("terrain_blob", params) + `]`
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

const snowPatch = `{"step":"terrain_blob","params":{"terrain_id":21,"seed_count":1,"size":7,"min_spacing":0,"placement":{"pattern":"ring","distance":[0,0]}}}`

func TestTerrainFillOnlyPaintsMatchingTerrain(t *testing.T) {
	dirt := step("terrain_fill", `{"terrain_id":9,"on":{"terrain_ids":[1]}}`)
	useScripts(t, map[string]string{"filtered": `[` + hexagon3 + `,` + grass + `,` + snowPatch + `,` + dirt + `]`})
	board := generateStartless(t, "filtered", 2)
	if snow, painted := len(withTerrain(board, 21)), len(withTerrain(board, 9)); snow != 7 || painted != 30 {
		t.Errorf("%d snow and %d dirt spaces, want the 7 snow kept and the other 30 painted", snow, painted)
	}
}

func TestTerrainBlobOnlySeedsAndGrowsOverMatchingTerrain(t *testing.T) {
	dirt := step("terrain_blob", `{"terrain_id":9,"seed_count":1,"size":200,"min_spacing":0,"on":{"terrain_names":["Snow"]}}`)
	useScripts(t, map[string]string{"filtered": `[` + hexagon3 + `,` + grass + `,` + snowPatch + `,` + dirt + `]`})
	board := generateStartless(t, "filtered", 2)
	if dirt, snow, grass := len(withTerrain(board, 9)), len(withTerrain(board, 21)), len(withTerrain(board, 1)); dirt != 7 || snow != 0 || grass != 30 {
		t.Errorf("%d dirt, %d snow, %d grass, want the whole snow patch (7) repainted and nothing else", dirt, snow, grass)
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
	useScripts(t, map[string]string{"fill": `[` + hexagon3 + `,` + grass + `]`})
	if got := len(withTerrain(generateStartless(t, "fill", 2), 1)); got != 37 {
		t.Errorf("%d grass spaces, want all 37", got)
	}
}

func TestVariantBlobConvertsOnlyToTheHillVariant(t *testing.T) {
	blob := `{"variant":"hilly","seed_count":1,"size":3,"min_spacing":0,"placement":{"pattern":"ring","distance":[0,0]}}`
	useScripts(t, map[string]string{"hills": blobScript(hexagon3, blob)})
	board := generateStartless(t, "hills", 2)
	if hills, grass := len(withTerrain(board, 51)), len(withTerrain(board, 1)); hills != 3 || grass != 34 {
		t.Errorf("%d hill and %d grass spaces, want 3 and 34", hills, grass)
	}
}

func TestMountainVariantOnlyGrowsOverHills(t *testing.T) {
	hills := `{"terrain_id":51,"seed_count":1,"size":7,"min_spacing":0,"placement":{"pattern":"ring","distance":[0,0]}}`
	mountains := step("terrain_blob", `{"variant":"mountainous","seed_count":1,"size":99,"min_spacing":0,"placement":{"pattern":"ring","distance":[0,0]},"on":{"terrain_types":["hilly"]}}`)
	useScripts(t, map[string]string{"peaks": `[` + hexagon3 + `,` + grass + `,` + step("terrain_blob", hills) + `,` + mountains + `]`})
	board := generateStartless(t, "peaks", 2)
	if peaks, hills, grass := len(withTerrain(board, 101)), len(withTerrain(board, 51)), len(withTerrain(board, 1)); peaks != 7 || hills != 0 || grass != 30 {
		t.Errorf("%d mountains, %d hills, %d grass, want the 7 hill spaces converted and the rest untouched", peaks, hills, grass)
	}
}

func TestVariantCannotCombineWithPaintedTerrain(t *testing.T) {
	bad := step("terrain_blob", `{"variant":"hilly","terrain_id":21,"seed_count":1,"size":3}`)
	useScripts(t, map[string]string{"bad": `[` + hexagon3 + `,` + grass + `,` + bad + `]`})
	if _, err := fakeboard.Generate("script:bad", 2, 1); err == nil || !strings.Contains(err.Error(), "variant can't be combined") {
		t.Errorf("got %v, want a variant exclusivity error", err)
	}
}

func TestTerrainFilterMatchesWholeFamilyAndAcceptsExpressions(t *testing.T) {
	hills := step("terrain_blob", `{"terrain_id":51,"seed_count":1,"size":3,"min_spacing":0,"placement":{"pattern":"ring","distance":[0,0]}}`)
	fill := step("terrain_fill", `{"terrain_id":21,"on":{"terrain_ids":["flat_id"]}}`)
	useScripts(t, map[string]string{"family": `[` + hexagon3 + `,` + step("define", `{"name":"flat_id","value":1}`) + `,` + grass + `,` + hills + `,` + fill + `]`})
	if got := len(withTerrain(generateStartless(t, "family", 2), 21)); got != 37 {
		t.Errorf("%d snow spaces, want all 37 because a flat id also matches its hill variant", got)
	}
}
