package risq_mapgen_tests

import (
	"math/rand"
	"reflect"
	"testing"

	"github.com/dgray001/gray_online/game/games/risq/internal/defs"
	"github.com/dgray001/gray_online/game/games/risq/internal/mapgen"
	"github.com/dgray001/gray_online/game/games/risq/tests/harness/fakeboard"
)

func resourceLevelBoard(t *testing.T, name string, level defs.MapResourceLevel) *fakeboard.Board {
	t.Helper()
	board := fakeboard.New()
	settings := mapgen.ResolveSettings(map[string]any{"map_resources": float64(level)}, 4)
	if err := mapgen.Generate(board, rand.New(rand.NewSource(1)), 4, settings, "script:"+name); err != nil {
		t.Fatal(err)
	}
	return board
}

func stoneCounts(board *fakeboard.Board) map[uint32]int {
	counts := make(map[uint32]int)
	for _, space := range board.Inspect() {
		for _, zone := range space.Zones {
			if zone.Resource >= 41 && zone.Resource <= 43 {
				counts[zone.Resource]++
			}
		}
	}
	return counts
}

func TestStoneTotalsOnEveryScriptMap(t *testing.T) {
	useTestScriptMaps(t)
	for level, want := range map[defs.MapResourceLevel]map[uint32]int{
		defs.MapResourceLevel_SPARSE:   {41: 6, 42: 6, 43: 1},
		defs.MapResourceLevel_STANDARD: {41: 8, 42: 8, 43: 6},
		defs.MapResourceLevel_RICH:     {41: 16, 42: 20, 43: 8},
		defs.MapResourceLevel_ABUNDANT: {41: 16, 42: 24, 43: 20},
	} {
		for _, name := range []string{"hexagon", "ring", "rectangle", "triangle"} {
			if got := stoneCounts(resourceLevelBoard(t, name, level)); !reflect.DeepEqual(got, want) {
				t.Errorf("%s resource level %d: stone nodes %v, want %v", name, level, got, want)
			}
		}
	}
}
