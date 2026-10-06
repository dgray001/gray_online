package risq_mapgen_tests

import (
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

func TestResourceClusterGrowsNearItsSeeds(t *testing.T) {
	useScripts(t, map[string]string{"cluster": `[` + hexagon3 + `,` + step("resource_cluster", `{"resource_id":43,"seed_count":2,"size":4,"zone":"edge"}`) + `]`})
	board := generateStartless(t, "cluster", 2)
	if got := len(resourceZones(board, 43)); got < 2 || got > 2*4 {
		t.Errorf("%d clustered zones, want between 2 and 8 for 2 seeds of size 4", got)
	}
	if len(board.Violations) > 0 {
		t.Errorf("violations: %v", board.Violations)
	}
}

func TestResourceMinSpacingThinsCloseResources(t *testing.T) {
	useScripts(t, map[string]string{"spaced": `[` + hexagon3 + `,` + step("resource_place", `{"resource_id":41,"count":30,"zone":"center"}`) + `,` + step("resource_min_spacing", `{"distance":3}`) + `]`})
	zones := resourceZones(generateStartless(t, "spaced", 2), 41)
	if len(zones) == 0 || len(zones) >= 30 {
		t.Fatalf("%d resources left of 30, want some but fewer", len(zones))
	}
	for i, a := range zones {
		for _, b := range zones[i+1:] {
			if game_utils.AxialDistance(a.space, b.space) < 3 {
				t.Errorf("resources in %v and %v are closer than 3 spaces", a.space, b.space)
			}
		}
	}
}

func TestResourceStepsRejectBadParams(t *testing.T) {
	cases := map[string]struct{ step, want string }{
		"unknown_id":   {step("resource_place", `{"resource_id":9999,"count":1}`), "unknown resource id"},
		"bad_zone":     {step("resource_place", `{"resource_id":41,"count":1,"zone":"corner"}`), "invalid zone"},
		"too_many":     {step("resource_place", `{"resource_id":41,"count":9999,"zone":"center"}`), "free zones"},
		"needs_starts": {step("resource_place", `{"resource_id":41,"count":1,"outside_player_areas":true}`), "needs an earlier player_starts"},
		"cluster_id":   {step("resource_cluster", `{"resource_id":9999,"seed_count":1,"size":1}`), "unknown resource id"},
		"cluster_zone": {step("resource_cluster", `{"resource_id":41,"seed_count":1,"size":1,"zone":"corner"}`), "invalid zone"},
		"scatter_cat":  {step("resource_scatter", `{"chance":1,"category_weights":{"gems":1}}`), "unknown resource_scatter category"},
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
