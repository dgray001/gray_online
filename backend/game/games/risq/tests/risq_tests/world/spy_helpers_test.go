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
	fakeboard.UseConfig(t, testConfig, nil, map[string]string{"spy": mapDoc("", spaces)})
	config := defs.UnitConfigs[villager]
	config.Vision.Space, config.Vision.Adjacent = defs.VisibilitySpy, defs.VisibilitySpy
	defs.UnitConfigs[15] = config
	return harness.NewGame(t, "custom:spy", 1, 2)
}

func spyTravelGame(t *testing.T) *harness.Game {
	t.Helper()
	spaces := spaceWith(0, 0, grass, unit(15, 0)) + "," + spaceWith(1, 0, grass, "") + "," + spaceWith(2, 0, grass, "") + "," + spaceWith(3, 0, grass, unit(villager, 1))
	fakeboard.UseConfig(t, testConfig, nil, map[string]string{"spy-travel": mapDoc("", spaces)})
	spy := defs.UnitConfigs[villager]
	spy.Vision.Space, spy.Vision.Adjacent = defs.VisibilitySpy, defs.VisibilitySpy
	defs.UnitConfigs[15] = spy
	worker := defs.UnitConfigs[villager]
	worker.Turn_stamina = 30
	defs.UnitConfigs[villager] = worker
	return harness.NewGame(t, "custom:spy-travel", 1, 2)
}

func spyHiddenTargetGame(t *testing.T) *harness.Game {
	t.Helper()
	spaces := spaceWith(0, 0, grass, unit(15, 0)) + "," + spaceWith(1, 0, grass, unit(11, 1)) + "," + spaceWith(2, 0, grass, unit(13, 2))
	doc := `{"board_size":3,"players":3,"spaces":[` + spaces + `]}`
	fakeboard.UseConfig(t, testConfig, nil, map[string]string{"hidden-target": doc})
	spy := defs.UnitConfigs[villager]
	spy.Vision.Space, spy.Vision.Adjacent = defs.VisibilitySpy, defs.VisibilitySpy
	defs.UnitConfigs[15] = spy
	attacker := defs.UnitConfigs[11]
	attacker.Attack_range = defs.RisqRange_ADJACENT
	attacker.Vision.Adjacent = defs.VisibilityGood
	defs.UnitConfigs[11] = attacker
	return harness.NewGame(t, "custom:hidden-target", 1, 3)
}

func spyGarrisonGame(t *testing.T) *harness.Game {
	t.Helper()
	spaces := spaceWith(0, 0, grass, unit(15, 0)) + `,{"x":1,"y":0,"terrain":1,"zones":[{"x":0,"y":0,"building":{"id":1,"player":1},"units":[{"id":1,"player":1,"count":1}]}]}`
	fakeboard.UseConfig(t, testConfig, nil, map[string]string{"spy-garrison": mapDoc("", spaces)})
	spy := defs.UnitConfigs[villager]
	spy.Vision.Space, spy.Vision.Adjacent = defs.VisibilitySpy, defs.VisibilitySpy
	defs.UnitConfigs[15] = spy
	return harness.NewGame(t, "custom:spy-garrison", 1, 2)
}
