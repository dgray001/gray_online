package risq_mapgen_tests

import (
	"fmt"
	"math/rand"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/dgray001/gray_online/game/games/risq/internal/defs"
	"github.com/dgray001/gray_online/game/games/risq/internal/mapgen"
	"github.com/dgray001/gray_online/game/games/risq/tests/harness/fakeboard"
)

func balancedBoard(t *testing.T, name string, balance any, players int, seed int64) *fakeboard.Board {
	t.Helper()
	board := fakeboard.New()
	settings := mapgen.ResolveSettings(map[string]any{"start_balance": balance}, players)
	if err := mapgen.Generate(board, rand.New(rand.NewSource(seed)), players, settings, "script:"+name); err != nil {
		t.Fatal(err)
	}
	return board
}

func useTestScriptMaps(t *testing.T) {
	t.Helper()
	scripts := make(map[string]string)
	for _, name := range []string{"hexagon", "ring", "rectangle", "triangle"} {
		data, err := os.ReadFile(filepath.Join(testConfig, "maps", "scripted", "fixture_"+name+".json"))
		if err != nil {
			t.Fatal(err)
		}
		scripts[name] = string(data)
	}
	useScripts(t, scripts)
}

func TestStartBalanceOnEveryScriptMap(t *testing.T) {
	t.Parallel()
	for _, name := range scriptPair(1) {
		t.Run(name, func(t *testing.T) {
			default_board := balancedBoard(t, name, nil, 4, 1)
			mirrored := balancedBoard(t, name, float64(defs.StartBalance_MIRRORED), 4, 1)
			random := balancedBoard(t, name, float64(defs.StartBalance_RANDOM), 4, 1)
			if default_board.Dump() != mirrored.Dump() {
				t.Error("default start balance differs from Mirrored")
			}
			if random.Dump() == mirrored.Dump() {
				t.Error("Random start balance did not change generation")
			}
			if name == "hexagon" && random.Dump() != balancedBoard(t, name, float64(defs.StartBalance_RANDOM), 4, 1).Dump() {
				t.Error("Random start balance is not reproducible")
			}
		})
	}
}

func TestRandomStartsKeepConfiguredAssets(t *testing.T) {
	t.Parallel()
	for _, name := range scriptPair(2) {
		for _, players := range []int{12} {
			board := balancedBoard(t, name, float64(defs.StartBalance_RANDOM), players, 1)
			for player := 0; player < players; player++ {
				units, buildings, _ := board.Tally(player)
				if units[11] != 3 || units[21] != 1 || buildings[201] != 1 {
					t.Errorf("%s %d players player %d: units %v buildings %v", name, players, player, units, buildings)
				}
			}
		}
	}
}

func TestExplicitStartBalanceOverridesGameSetting(t *testing.T) {
	params := strings.TrimSuffix(nomadTestStarts, "}") + fmt.Sprintf(`,"start_balance":%d}`, defs.StartBalance_MIRRORED)
	useScripts(t, map[string]string{"override": `[` + hexagon2 + `,` + step("player_starts", params) + `]`})
	random := balancedBoard(t, "override", float64(defs.StartBalance_RANDOM), 4, 1)
	mirrored := balancedBoard(t, "override", float64(defs.StartBalance_MIRRORED), 4, 1)
	if random.Dump() != mirrored.Dump() {
		t.Error("explicit Mirrored parameter did not override Random game setting")
	}
}
