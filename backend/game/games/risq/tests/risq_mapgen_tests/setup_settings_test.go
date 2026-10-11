package risq_mapgen_tests

import (
	"testing"

	"github.com/dgray001/gray_online/game/games/risq/internal/defs"
	"github.com/dgray001/gray_online/game/games/risq/tests/harness"
)

func TestScriptsReceiveSetupSettings(t *testing.T) {
	for key, values := range map[string][]float64{
		"starting_stockpile":   {float64(defs.StartingStockpile_LOW), float64(defs.StartingStockpile_DEATHMATCH)},
		"map_resources":        {float64(defs.MapResourceLevel_SPARSE), float64(defs.MapResourceLevel_ABUNDANT)},
		"starting_development": {float64(defs.StartingDevelopment_NOMAD), float64(defs.StartingDevelopment_ADVANCED)},
		"start_balance":        {float64(defs.StartBalance_RANDOM), float64(defs.StartBalance_MIRRORED)},
	} {
		t.Run(key, func(t *testing.T) {
			useScripts(t, map[string]string{"setting": `[` + hexagon2 + `,{"step":"rules","params":{"space_gold_income":"` + key + `"}},{"step":"player_starts","params":{"pattern":"ring","starting_distance":1,"area_size":1,"terrain_id":1,"assets":{"units":[{"unit_id":1,"count":1,"target":{"zone":"center"}}]},"resources":[]}}]`})
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
