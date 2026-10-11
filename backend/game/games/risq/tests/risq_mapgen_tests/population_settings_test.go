package risq_mapgen_tests

import (
	"math"
	"testing"

	"github.com/dgray001/gray_online/game"
	"github.com/dgray001/gray_online/game/games/risq"
	"github.com/dgray001/gray_online/game/games/risq/tests/harness"
)

func TestPopulationLimitSettingReachesPlayers(t *testing.T) {
	for raw, want := range map[any]int{nil: 100, float64(25): 25, float64(100): 100, float64(200): 200} {
		g := harness.NewUnstartedGameWithSettings(t, "script:hexagon", 1, 2, map[string]any{"population_limit": raw})
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
		base := game.CreateBaseGame(1, game.GameType_RISQ, map[string]any{"population_limit": raw})
		if _, err := risq.CreateGame(base, make(chan game.PlayerAction, 1)); err == nil || err.Error() != "population_limit must be an integer between 25 and 200" {
			t.Errorf("population_limit %v: got error %v", raw, err)
		}
	}
}
