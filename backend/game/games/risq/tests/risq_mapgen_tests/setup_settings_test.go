package risq_mapgen_tests

import (
	"math"
	"math/rand"
	"testing"

	"github.com/dgray001/gray_online/game"
	"github.com/dgray001/gray_online/game/games/risq"
	"github.com/dgray001/gray_online/game/games/risq/internal/defs"
	"github.com/dgray001/gray_online/game/games/risq/internal/mapgen"
	"github.com/dgray001/gray_online/game/games/risq/tests/harness"
	"github.com/dgray001/gray_online/game/games/risq/tests/harness/fakeboard"
)

func TestScriptsReceiveSetupSettings(t *testing.T) {
	for key, values := range map[string][]float64{
		"starting_stockpile":   {float64(defs.StartingStockpile_LOW), float64(defs.StartingStockpile_DEATHMATCH)},
		"map_resources":        {float64(defs.MapResourceLevel_SPARSE), float64(defs.MapResourceLevel_ABUNDANT)},
		"starting_development": {float64(defs.StartingDevelopment_NOMAD), float64(defs.StartingDevelopment_ADVANCED)},
		"start_balance":        {float64(defs.StartBalance_RANDOM), float64(defs.StartBalance_MIRRORED)},
	} {
		t.Run(key, func(t *testing.T) {
			useScripts(t, map[string]string{"setting": `[` + hexagon2 + `,{"step":"rules","params":{"space_gold_income":"` + key + `"}},{"step":"player_starts","params":{"pattern":"ring","starting_distance":1,"area_size":1,"terrain_id":1,"assets":{"units":[{"unit_id":11,"count":1,"target":{"zone":"center"}}]},"resources":[]}}]`})
			for _, value := range values {
				g := harness.NewUnstartedGameWithSettings(t, "script:setting", 1, 2, map[string]any{key: value})
				g.Risq.StartGame()
				space := g.Self(0).Units[0].Space
				if got := g.State(0).Space(space.X, space.Y).GoldIncome; got != value {
					t.Errorf("%s: script received %v, want %v", key, got, value)
				}
			}
		})
	}
}

func TestPopulationLimitSettingReachesPlayers(t *testing.T) {
	for raw, want := range map[any]int{nil: 100, float64(25): 25, float64(100): 100, float64(200): 200} {
		g := harness.NewUnstartedGameWithSettings(t, "script:fixture_hexagon", 1, 2, map[string]any{"population_limit": raw})
		g.Risq.StartGame()
		for player := 0; player < 2; player++ {
			if got := g.Self(player).MaxPopulationLimit; got != want {
				t.Errorf("population_limit %v player %d: got %d, want %d", raw, player, got, want)
			}
		}
	}
}

func TestPopulationLimitRejectsInvalidSettings(t *testing.T) {
	for _, raw := range []any{float64(0), float64(24), float64(201), 25.5, "100", math.NaN(), math.Inf(1)} {
		base := game.CreateBaseGame(1, game.GameType_RISQ, map[string]any{"population_limit": raw}, nil, nil)
		if _, err := risq.CreateGame(base, make(chan game.PlayerAction, 1)); err == nil || err.Error() != "population_limit must be an integer between 25 and 200" {
			t.Errorf("population_limit %v: got error %v", raw, err)
		}
	}
}

func TestCreateGameRejectsUnlistedMapAndAiConfig(t *testing.T) {
	for _, settings := range []map[string]any{
		aiSettings(2, 1, "script:missing"),
		aiSettings(2, 1, "custom:../../ai/default"),
		withAiConfig(aiSettings(2, 1, "script:fixture_ring"), "../maps/scripted/fixture_ring"),
	} {
		base := game.CreateBaseGame(1, game.GameType_RISQ, settings, nil, nil)
		if _, err := risq.CreateGame(base, make(chan game.PlayerAction, 1)); err == nil {
			t.Errorf("settings %v should be rejected", settings)
		}
	}
	base := game.CreateBaseGame(1, game.GameType_RISQ, withAiConfig(aiSettings(2, 1, "script:fixture_ring"), "temp_retreat"), nil, nil)
	if _, err := risq.CreateGame(base, make(chan game.PlayerAction, 1)); err != nil {
		t.Errorf("listed ai config rejected: %v", err)
	}
}

func withAiConfig(settings map[string]any, config string) map[string]any {
	for _, ai := range settings["ai_players"].([]any) {
		ai.(map[string]any)["config"] = config
	}
	return settings
}

func createWithAis(count int) error {
	_, err := engineStarts("script:fixture_ring", count, 1)
	return err
}

func TestCreateGameRejectsInvalidPlayerCounts(t *testing.T) {
	for _, count := range []int{0, 1, 13} {
		if err := createWithAis(count); err == nil {
			t.Errorf("%d players should be rejected", count)
		}
	}
	for _, count := range []int{2, 12} {
		if err := createWithAis(count); err != nil {
			t.Errorf("%d players should be accepted: %v", count, err)
		}
	}
}

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
