package invariants

import (
	"fmt"
	"testing"

	"github.com/dgray001/gray_online/game/games/risq/tests/harness"
	"github.com/dgray001/gray_online/game/games/risq/tests/harness/fakeboard"
)

func invariantGame(t *testing.T, players int, spaces, extra string, configure func()) *harness.Game {
	t.Helper()
	doc := fmt.Sprintf(`{"board_size":1,"players":%d,"starting_bank":{"food":500,"wood":500,"stone":500,"gold":500},"spaces":[%s]%s}`, players, spaces, extra)
	fakeboard.UseConfig(t, "../../../config", nil, map[string]string{"invariants": doc})
	configure()
	allSeeingConfig()
	return harness.NewGame(t, "custom:invariants", 37, players)
}
