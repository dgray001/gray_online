package orders

import (
	"testing"

	"github.com/dgray001/gray_online/game/games/risq/internal/defs"
	"github.com/dgray001/gray_online/game/games/risq/tests/harness"
	"github.com/dgray001/gray_online/game/games/risq/tests/harness/fakeboard"
)

func notificationGame(t *testing.T, spy bool) *harness.Game {
	t.Helper()
	doc := `{"board_size":3,"players":2,"spaces":[{"x":0,"y":0,"terrain":1,"zones":[{"x":0,"y":0,"building":{"id":23,"player":0},"units":[{"id":11,"player":0,"count":1},{"id":1,"player":1,"count":1}]}]},{"x":3,"y":0,"terrain":1,"zones":[{"x":0,"y":0,"building":{"id":21,"player":0},"units":[{"id":11,"player":0,"count":1}]}]}]}`
	fakeboard.UseConfig(t, testConfig, nil, map[string]string{"notifications": doc})
	config := defs.UnitConfigs[1]
	config.Vision.Space = defs.VisibilityGood
	if spy {
		config.Vision.Space = defs.VisibilitySpy
	}
	defs.UnitConfigs[1] = config
	return harness.NewGame(t, "custom:notifications", 1, 2)
}
