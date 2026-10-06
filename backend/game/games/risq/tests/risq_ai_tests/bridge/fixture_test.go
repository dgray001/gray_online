package bridge

import (
	"github.com/dgray001/gray_online/game"
	"github.com/dgray001/gray_online/game/games/risq/tests/harness"
	"github.com/dgray001/gray_online/game/games/risq/tests/harness/fakeboard"
	"testing"
)

func bridgeGame(t *testing.T, spaces string, configure ...func()) (*harness.Game, *game.Player) {
	t.Helper()
	document := `{"board_size":4,"players":2,"spaces":[` + spaces + `]}`
	fakeboard.UseConfig(t, "../../../config", nil, map[string]string{"bridge-test": document})
	for _, apply := range configure {
		apply()
	}
	g := harness.NewGame(t, "custom:bridge-test", 1, 2)
	player := game.CreateAiPlayer("probe", g.Base)
	player.Player_id = 0
	return g, player
}
