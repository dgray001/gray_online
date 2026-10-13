package invariants

import (
	"fmt"
	"sync/atomic"
	"testing"

	"github.com/dgray001/gray_online/game/games/risq/tests/config"
	"github.com/dgray001/gray_online/game/games/risq/tests/harness"
	"github.com/dgray001/gray_online/game/games/risq/tests/harness/fakeboard"
)

// Set while parallel runs share the config the scenario's first game applied
var configShared atomic.Bool

func invariantGame(t *testing.T, players int, spaces, extra string, configure func()) *harness.Game {
	t.Helper()
	if !configShared.Load() {
		doc := fmt.Sprintf(`{"board_size":1,"players":%d,"starting_bank":{"food":500,"wood":500,"stone":500,"gold":500},"spaces":[%s]%s}`, players, spaces, extra)
		fakeboard.UseConfig(t, testconfig.Dir(), nil, map[string]string{"invariants": doc})
		configure()
		allSeeingConfig()
	}
	g := harness.NewGame(t, "custom:invariants", 37, players)
	g.Lite = true
	return g
}
