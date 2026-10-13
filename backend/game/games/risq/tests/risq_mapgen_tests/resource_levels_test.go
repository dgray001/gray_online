package risq_mapgen_tests

import (
	"math/rand"
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

func TestScriptReceivesMapResources(t *testing.T) {
	useScripts(t, map[string]string{"leveled": `[` + hexagon2 + `,{"step":"rules","params":{"space_gold_income":"map_resources"}}]`})
	for _, level := range []defs.MapResourceLevel{defs.MapResourceLevel_SPARSE, defs.MapResourceLevel_ABUNDANT} {
		board := fakeboard.New()
		settings := mapgen.ResolveSettings(map[string]any{"map_resources": float64(level)}, 4)
		if err := mapgen.Generate(board, rand.New(rand.NewSource(1)), 4, settings, "script:leveled"); err == nil || err.Error() != startsMissing {
			t.Fatalf("level %d: error %v", level, err)
		}
		if got := goldIncome(t, board); got != float64(level) {
			t.Errorf("level %d: script received %v", level, got)
		}
	}
}
