package risq_mapgen_tests

import (
	"strconv"
	"testing"

	"github.com/dgray001/gray_online/game/game_utils"
	"github.com/dgray001/gray_online/game/games/risq/tests/harness/fakeboard"
)

const (
	hexagon3 = `{"step":"shape","params":{"kind":"hexagon","size":3}}`
	ring3    = `{"step":"shape","params":{"kind":"ring","size":3,"inner_size":1}}`
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

func TestTerrainFillPaintsEverySpace(t *testing.T) {
	useScripts(t, map[string]string{"fill": `[` + hexagon3 + `,` + grass + `]`})
	if got := len(withTerrain(generateStartless(t, "fill", 2), 1)); got != 37 {
		t.Errorf("%d grass spaces, want all 37", got)
	}
}

func TestTerrainLineDrawsAStripAlongTheLine(t *testing.T) {
	line := func(width int) string {
		return `[` + hexagon3 + `,` + grass + `,{"step":"terrain_line","params":{"terrain_id":51,"width":` + strconv.Itoa(width) + `,"from":{"x":-3,"y":0},"to":{"x":3,"y":0}}}]`
	}
	useScripts(t, map[string]string{"line1": line(1), "line2": line(2)})
	thin := withTerrain(generateStartless(t, "line1", 2), 51)
	if len(thin) != 7 {
		t.Fatalf("width 1 painted %d spaces, want the 7 on the line", len(thin))
	}
	for _, c := range thin {
		if c.Y != 0 {
			t.Errorf("width 1 painted %v off the line", c)
		}
	}
	if wide := withTerrain(generateStartless(t, "line2", 2), 51); len(wide) <= 7 {
		t.Errorf("width 2 painted %d spaces, want more than the 7 on the line", len(wide))
	}
}

func TestTerrainBorderPaintsFromTheChosenEdge(t *testing.T) {
	border := func(shape string, edge string, depth int) string {
		return `[` + shape + `,` + grass + `,{"step":"terrain_border","params":{"terrain_id":51,"edge":"` + edge + `","depth":` + strconv.Itoa(depth) + `}}]`
	}
	useScripts(t, map[string]string{
		"hex0": border(hexagon3, "hexagon_edge", 0), "hex1": border(hexagon3, "hexagon_edge", 1),
		"map0": border(hexagon3, "map_edge", 0), "ringhex": border(ring3, "hexagon_edge", 0), "ringmap": border(ring3, "map_edge", 0),
	})
	want := map[string]int{"hex0": 18, "hex1": 30, "map0": 18, "ringhex": 18, "ringmap": 30}
	for name, count := range want {
		if got := len(withTerrain(generateStartless(t, name, 2), 51)); got != count {
			t.Errorf("%s painted %d spaces, want %d", name, got, count)
		}
	}
}
