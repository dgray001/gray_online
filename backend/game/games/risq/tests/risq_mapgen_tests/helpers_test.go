package risq_mapgen_tests

import (
	"fmt"
	"testing"

	"github.com/dgray001/gray_online/game/games/risq/tests/harness/fakeboard"
)

const startsMissing = "map script did not place all player starts"

func useScripts(t *testing.T, scripts map[string]string) {
	t.Helper()
	fakeboard.UseConfig(t, testConfig, scripts, nil)
}

func useCustoms(t *testing.T, customs map[string]string) {
	t.Helper()
	fakeboard.UseConfig(t, testConfig, nil, customs)
}

// Runs a script that places no players, accepting only the final "no starts" error so the board can still be inspected
func generateStartless(t *testing.T, name string, players int) *fakeboard.Board {
	t.Helper()
	board, err := fakeboard.Generate("script:"+name, players, 1)
	if err == nil || err.Error() != startsMissing {
		t.Fatalf("script %s: got error %v, want %q", name, err, startsMissing)
	}
	return board
}

// Game settings for an all-AI game of the given map; the engine builds its players from these
func aiSettings(count int, seed int64, mapName string) map[string]any {
	ais := make([]any, count)
	for i := range ais {
		ais[i] = map[string]any{"nickname": fmt.Sprint("ai", i)}
	}
	return map[string]any{"ai_players": ais, "seed": float64(seed), "map": mapName}
}

// The space gold income a board was given, failing the test when none was set
func goldIncome(t *testing.T, board *fakeboard.Board) float64 {
	t.Helper()
	if board.GoldIncome == nil {
		t.Fatal("no space gold income was set")
	}
	return *board.GoldIncome
}

func shapeScript(params string) string {
	return `[{"step":"shape","params":` + params + `}]`
}
