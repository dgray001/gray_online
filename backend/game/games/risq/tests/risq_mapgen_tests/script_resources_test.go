package risq_mapgen_tests

import (
	"strconv"
	"strings"
	"testing"

	"github.com/dgray001/gray_online/game/game_utils"
	"github.com/dgray001/gray_online/game/games/risq/tests/harness/fakeboard"
)

func step(name string, params string) string {
	return `{"step":"` + name + `","params":` + params + `}`
}

type resourceZone struct {
	space  game_utils.Coordinate2D
	local  game_utils.Coordinate2D
	center bool
}

// Every zone holding the given resource id
func resourceZones(board *fakeboard.Board, resource uint32) []resourceZone {
	var zones []resourceZone
	for _, info := range board.Inspect() {
		for _, z := range info.Zones {
			if z.Resource == resource {
				zones = append(zones, resourceZone{info.Coord, z.Local, z.Local == game_utils.Coordinate2D{}})
			}
		}
	}
	return zones
}

func TestResourcePlaceDropsExactlyCountOnMatchingZones(t *testing.T) {
	useScripts(t, map[string]string{"place": `[` + hexagon3 + `,` + step("resource_place", `{"resource_id":41,"count":25,"zone":"edge"}`) + `]`})
	board := generateStartless(t, "place", 2)
	zones := resourceZones(board, 41)
	if len(zones) != 25 {
		t.Fatalf("%d resources, want exactly 25", len(zones))
	}
	for _, z := range zones {
		if z.center {
			t.Errorf("edge-only resource landed on the center zone of %v", z.space)
		}
	}
	if len(board.Violations) > 0 {
		t.Errorf("violations: %v", board.Violations)
	}
}

func TestResourcePlaceClumpsKeepTheTotalCount(t *testing.T) {
	useScripts(t, map[string]string{"clump": `[` + hexagon3 + `,` + step("resource_place", `{"resource_id":43,"count":8,"clump_size":4,"zone":"edge"}`) + `]`})
	board := generateStartless(t, "clump", 2)
	if got := len(resourceZones(board, 43)); got != 8 {
		t.Errorf("%d clumped resources, want exactly the count of 8", got)
	}
	if len(board.Violations) > 0 {
		t.Errorf("violations: %v", board.Violations)
	}
}

func TestResourcePlaceKeepsMinSpacingWhileItCan(t *testing.T) {
	useScripts(t, map[string]string{"spaced": `[` + hexagon3 + `,` + step("resource_place", `{"resource_id":41,"count":3,"zone":"center","min_spacing":3}`) + `]`})
	zones := resourceZones(generateStartless(t, "spaced", 2), 41)
	if len(zones) != 3 {
		t.Fatalf("%d resources, want 3", len(zones))
	}
	for i, a := range zones {
		for _, b := range zones[i+1:] {
			if game_utils.AxialDistance(a.space, b.space) < 3 {
				t.Errorf("resources in %v and %v are closer than 3 spaces", a.space, b.space)
			}
		}
	}
}

func TestResourcePlaceFallsBackUntilNoFreeZoneRemains(t *testing.T) {
	useScripts(t, map[string]string{"limited": `[` + hexagon3 + `,` + step("resource_place", `{"resource_id":41,"count":9999,"zone":"center"}`) + `,` + step("resource_place", `{"resource_id":43,"count":9999,"zone":"center"}`) + `]`})
	board := generateStartless(t, "limited", 2)
	if got, want := len(resourceZones(board, 41)), 7*len(board.Spaces()); got != want {
		t.Errorf("placed %d small mines, want every one of the %d zones filled", got, want)
	}
	if got := len(resourceZones(board, 43)); got != 0 {
		t.Errorf("placed %d large mines with no free zone left", got)
	}
}

func TestResourcePlaceTerrainFilterPrefersButDoesNotRequire(t *testing.T) {
	hillyTerrain := step("terrain_blob", `{"terrain_id":51,"seed_count":1,"size":3,"min_spacing":0,"placement":{"pattern":"ring","distance":[0,0]}}`)
	place := func(count int) string {
		return `[` + hexagon3 + `,` + flatTerrain + `,` + hillyTerrain + `,` + step("resource_place", `{"resource_id":41,"count":`+strconv.Itoa(count)+`,"on":{"terrain_types":["hilly"]}}`) + `]`
	}
	useScripts(t, map[string]string{"few": place(10), "many": place(40)})
	for name, count := range map[string]int{"few": 10, "many": 40} {
		board := generateStartless(t, name, 2)
		on_hills := 0
		for _, info := range board.Inspect() {
			for _, z := range info.Zones {
				if z.Resource == 41 && info.Terrain == 51 {
					on_hills++
				}
			}
		}
		if got := len(resourceZones(board, 41)); got != count {
			t.Errorf("%s: placed %d, want %d", name, got, count)
		}
		if want := min(count, 21); on_hills != want {
			t.Errorf("%s: %d on hilly terrain, want %d (every hilly zone before any other)", name, on_hills, want)
		}
	}
}

func TestResourceStepsRejectBadParams(t *testing.T) {
	cases := map[string]struct{ step, want string }{
		"unknown_id":   {step("resource_place", `{"resource_id":9999,"count":1}`), "unknown resource id"},
		"bad_zone":     {step("resource_place", `{"resource_id":41,"count":1,"zone":"corner"}`), "invalid zone"},
		"needs_starts": {step("resource_place", `{"resource_id":41,"count":1,"outside_player_areas":true}`), "needs an earlier player_starts"},
		"no_resource":  {step("resource_place", `{"count":1}`), "needs a resource_id or positive weights"},
		"bad_terrain":  {step("resource_place", `{"resource_id":41,"count":1,"on":{"terrain_names":["Lava"]}}`), "unknown terrain name"},
		"blob_cat":     {step("resource_blob", `{"category_weights":{"gems":1},"seed_count":1,"size":1}`), "unknown resource category"},
	}
	scripts := map[string]string{}
	for name, c := range cases {
		scripts[name] = `[` + hexagon3 + `,` + c.step + `]`
	}
	useScripts(t, scripts)
	for name, c := range cases {
		if _, err := fakeboard.Generate("script:"+name, 2, 1); err == nil || !strings.Contains(err.Error(), c.want) {
			t.Errorf("%s: error %v, want it to contain %q", name, err, c.want)
		}
	}
}

func TestResourcePlaceEvennessSpreadsClumpsApart(t *testing.T) {
	place := func(evenness string) string {
		return `[` + hexagon3 + `,` + step("resource_place", `{"resource_id":41,"count":4,"zone":"center","evenness":`+evenness+`}`) + `]`
	}
	useScripts(t, map[string]string{"even": place("1"), "random": place("0")})
	closestPairs := func(name string) int {
		total := 0
		for seed := int64(1); seed <= 8; seed++ {
			board, _ := fakeboard.Generate("script:"+name, 2, seed)
			zones := resourceZones(board, 41)
			closest := 99
			for i, a := range zones {
				for _, b := range zones[i+1:] {
					closest = min(closest, int(game_utils.AxialDistance(a.space, b.space)))
				}
			}
			total += closest
		}
		return total
	}
	if even, random := closestPairs("even"), closestPairs("random"); even <= random {
		t.Errorf("closest-pair distances summed over 8 seeds: evenness 1 gave %d, evenness 0 gave %d, want evenness to spread clumps further apart", even, random)
	}
}

func TestResourcePlacePreferredSpacingRelaxesToTheHardFloor(t *testing.T) {
	useScripts(t, map[string]string{"relax": `[` + hexagon3 + `,` + step("resource_place", `{"resource_id":41,"count":6,"zone":"center","preferred_spacing":6,"min_spacing":2}`) + `]`})
	zones := resourceZones(generateStartless(t, "relax", 2), 41)
	if len(zones) != 6 {
		t.Fatalf("%d resources, want all 6 placed", len(zones))
	}
	closest := 99
	for i, a := range zones {
		for _, b := range zones[i+1:] {
			closest = min(closest, int(game_utils.AxialDistance(a.space, b.space)))
		}
	}
	if closest < 2 || closest >= 6 {
		t.Errorf("closest pair %d spaces apart, want at least the floor of 2 and below the unreachable preferred 6", closest)
	}
}

func TestResourcePlaceGivesUpTerrainBeforeZoneSelector(t *testing.T) {
	hillyTerrain := step("terrain_blob", `{"terrain_id":51,"seed_count":1,"size":3,"min_spacing":0,"placement":{"pattern":"ring","distance":[0,0]}}`)
	place := func(count int) string {
		return `[` + hexagon3 + `,` + flatTerrain + `,` + hillyTerrain + `,` + step("resource_place", `{"resource_id":41,"count":`+strconv.Itoa(count)+`,"zone":"center","on":{"terrain_types":["hilly"]}}`) + `]`
	}
	useScripts(t, map[string]string{"centers": place(10), "overflow": place(38)})
	for name, count := range map[string]int{"centers": 10, "overflow": 38} {
		zones := resourceZones(generateStartless(t, name, 2), 41)
		centers := 0
		for _, z := range zones {
			if z.center {
				centers++
			}
		}
		if len(zones) != count || centers != min(count, 37) {
			t.Errorf("%s: %d placed with %d on center zones, want %d and %d", name, len(zones), centers, count, min(count, 37))
		}
	}
}

func TestResourceBlobResolvesWeightKeysAsExpressions(t *testing.T) {
	blob := step("resource_blob", `{"resource_weights":{"base_tree":1,"base_tree + 10":1},"seed_count":4,"size":3,"min_spacing":0}`)
	useScripts(t, map[string]string{"weights": `[` + hexagon3 + `,` + step("define", `{"name":"base_tree","value":11}`) + `,` + blob + `]`})
	board := generateStartless(t, "weights", 2)
	if len(resourceZones(board, 11))+len(resourceZones(board, 21)) == 0 {
		t.Fatal("no resources of the expression-resolved ids 11 or 21 were placed")
	}
	for _, info := range board.Inspect() {
		for _, z := range info.Zones {
			if z.Resource != 0 && z.Resource != 11 && z.Resource != 21 {
				t.Errorf("unexpected resource id %d", z.Resource)
			}
		}
	}
}

func TestResourceBlobStaysOnMatchingTerrain(t *testing.T) {
	hillyTerrain := step("terrain_blob", `{"terrain_id":51,"seed_count":1,"size":3,"min_spacing":0,"placement":{"pattern":"ring","distance":[0,0]}}`)
	blob := step("resource_blob", `{"resource_id":11,"seed_count":1,"size":300,"on":{"terrain_types":["hilly"]}}`)
	useScripts(t, map[string]string{"filtered": `[` + hexagon3 + `,` + flatTerrain + `,` + hillyTerrain + `,` + blob + `]`})
	board := generateStartless(t, "filtered", 2)
	placed := 0
	for _, info := range board.Inspect() {
		for _, z := range info.Zones {
			if z.Resource == 11 {
				placed++
				if info.Terrain != 51 {
					t.Errorf("resource on terrain %d, want only hilly terrain", info.Terrain)
				}
			}
		}
	}
	if placed == 0 || placed > 21 {
		t.Errorf("placed %d resources, want between 1 and the 21 zones of the 3 hilly spaces", placed)
	}
}

func TestResourceBlobOverwriteReplacesExistingResources(t *testing.T) {
	fill := step("resource_place", `{"resource_id":41,"count":9999}`)
	script := func(overwrite string) string {
		return `[` + hexagon3 + `,` + fill + `,` + step("resource_blob", `{"resource_id":42,"seed_count":1,"size":5,"overwrite":`+overwrite+`}`) + `]`
	}
	useScripts(t, map[string]string{"keep": script("false"), "replace": script("true")})
	if got := len(resourceZones(generateStartless(t, "keep", 2), 42)); got != 0 {
		t.Errorf("without overwrite %d resources replaced existing ones, want 0", got)
	}
	replaced := generateStartless(t, "replace", 2)
	if got, total := len(resourceZones(replaced, 42)), 7*len(replaced.Spaces()); got != 5 || len(resourceZones(replaced, 41)) != total-5 {
		t.Errorf("with overwrite %d replaced, %d originals left of %d, want 5 and %d", got, len(resourceZones(replaced, 41)), total, total-5)
	}
}
