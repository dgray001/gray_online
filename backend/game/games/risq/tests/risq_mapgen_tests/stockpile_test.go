package risq_mapgen_tests

import (
	"testing"

	"github.com/dgray001/gray_online/game/games/risq/internal/defs"
	"github.com/dgray001/gray_online/game/games/risq/tests/harness"
)

func TestCustomStartingBankOverridesStockpile(t *testing.T) {
	useCustoms(t, map[string]string{"bank": customDoc(2, `,"starting_bank":{"food":5,"wood":6,"stone":7,"gold":8}`, hex7)})
	g := harness.NewUnstartedGameWithSettings(t, "custom:bank", 1, 2, map[string]any{"starting_stockpile": float64(defs.StartingStockpile_DEATHMATCH)})
	for player := 0; player < 2; player++ {
		if got, want := *g.Self(player).Resources, (harness.Resources{Food: 5, Wood: 6, Stone: 7, Gold: 8}); got != want {
			t.Errorf("player %d: map override got %+v, want %+v", player, got, want)
		}
	}
}

func TestScriptStartingBankOverridesStockpile(t *testing.T) {
	useScripts(t, map[string]string{"bank": `[` + hexagon2 + `,{"step":"player_starts","params":{"pattern":"ring","starting_distance":1,"area_size":1,"terrain_id":1,"starting_bank":{"food":5,"wood":6,"stone":7,"gold":8}}}]`})
	g := harness.NewUnstartedGameWithSettings(t, "script:bank", 1, 2, map[string]any{"starting_stockpile": float64(defs.StartingStockpile_DEATHMATCH)})
	for player := 0; player < 2; player++ {
		if got, want := *g.Self(player).Resources, (harness.Resources{Food: 5, Wood: 6, Stone: 7, Gold: 8}); got != want {
			t.Errorf("player %d: map override got %+v, want %+v", player, got, want)
		}
	}
}

func TestStartingStockpilesOnScriptMaps(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name      string
		stockpile defs.StartingStockpile
		want      harness.Resources
	}{
		{"hexagon", defs.StartingStockpile_LOW, harness.Resources{Food: 100, Wood: 100, Stone: 50}},
		{"ring", defs.StartingStockpile_STANDARD, harness.Resources{Food: 200, Wood: 200, Stone: 100}},
		{"rectangle", defs.StartingStockpile_HIGH, harness.Resources{Food: 500, Wood: 500, Stone: 250, Gold: 100}},
		{"triangle", defs.StartingStockpile_DEATHMATCH, harness.Resources{Food: 10000, Wood: 10000, Stone: 5000, Gold: 5000}},
	}
	for _, c := range cases {
		g := harness.NewUnstartedGameWithSettings(t, "script:"+c.name, 1, 2, map[string]any{"starting_stockpile": float64(c.stockpile)})
		g.Risq.StartGame()
		for player := 0; player < 2; player++ {
			if got := *g.Self(player).Resources; got != c.want {
				t.Errorf("%s stockpile %d player %d: got %+v, want %+v", c.name, c.stockpile, player, got, c.want)
			}
		}
	}
}
