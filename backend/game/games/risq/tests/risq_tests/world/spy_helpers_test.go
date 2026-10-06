package world

import (
	"testing"

	"github.com/dgray001/gray_online/game/games/risq/internal/defs"
	"github.com/dgray001/gray_online/game/games/risq/tests/harness"
	"github.com/dgray001/gray_online/game/games/risq/tests/harness/fakeboard"
)

func spyGame(t *testing.T) *harness.Game {
	t.Helper()
	spaces := `{"x":0,"y":0,"terrain":1,"zones":[{"x":0,"y":0,"building":{"id":21,"player":0},"units":[{"id":15,"player":0,"count":1}]},{"x":1,"y":0,"building":{"id":1,"player":0}}]},{"x":1,"y":0,"terrain":1,"zones":[{"x":0,"y":0,"resource":41,"units":[{"id":1,"player":1,"count":1}]}]}`
	fakeboard.UseConfig(t, shippedConfig, nil, map[string]string{"spy": mapDoc("", spaces)})
	config := defs.UnitConfigs[villager]
	config.Vision.Space, config.Vision.Adjacent = defs.VisibilitySpy, defs.VisibilitySpy
	defs.UnitConfigs[15] = config
	return harness.NewGame(t, "custom:spy", 1, 2)
}
