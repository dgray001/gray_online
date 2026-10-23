package world

import (
	"testing"

	"github.com/dgray001/gray_online/game/games/risq/internal/defs"
	"github.com/dgray001/gray_online/game/games/risq/tests/harness"
)

func spyGame(t *testing.T) *harness.Game {
	t.Helper()
	t.Cleanup(func() { pristine.Restore(baseDir) })
	spaces := `{"x":0,"y":0,"terrain":1,"zones":[{"x":0,"y":0,"building":{"id":207,"player":0},"units":[{"id":15,"player":0,"count":1}]},{"x":1,"y":0,"building":{"id":201,"player":0}}]},{"x":1,"y":0,"terrain":1,"zones":[{"x":0,"y":0,"resource":41,"units":[{"id":11,"player":1,"count":1}]}]}`
	writeCustomMap("spy", mapDoc("", spaces))
	config := defs.UnitConfigs[worker]
	config.Vision.Space, config.Vision.Adjacent = defs.VisibilitySpy, defs.VisibilitySpy
	defs.UnitConfigs[15] = config
	return harness.NewGame(t, "custom:spy", 1, 2)
}

func spyTravelGame(t *testing.T) *harness.Game {
	t.Helper()
	t.Cleanup(func() { pristine.Restore(baseDir) })
	spaces := spaceWith(0, 0, flatTerrain, unit(15, 0)) + "," + spaceWith(1, 0, flatTerrain, "") + "," + spaceWith(2, 0, flatTerrain, "") + "," + spaceWith(3, 0, flatTerrain, unit(worker, 1))
	writeCustomMap("spy-travel", mapDoc("", spaces))
	spy := defs.UnitConfigs[worker]
	spy.Vision.Space, spy.Vision.Adjacent = defs.VisibilitySpy, defs.VisibilitySpy
	defs.UnitConfigs[15] = spy
	fast := defs.UnitConfigs[worker]
	fast.Turn_stamina = 30
	defs.UnitConfigs[worker] = fast
	return harness.NewGame(t, "custom:spy-travel", 1, 2)
}

func spyHiddenTargetGame(t *testing.T) *harness.Game {
	t.Helper()
	t.Cleanup(func() { pristine.Restore(baseDir) })
	spaces := spaceWith(0, 0, flatTerrain, unit(15, 0)) + "," + spaceWith(1, 0, flatTerrain, unit(grunt, 1)) + "," + spaceWith(2, 0, flatTerrain, unit(22, 2))
	doc := `{"board_size":3,"players":3,"spaces":[` + spaces + `]}`
	writeCustomMap("hidden-target", doc)
	spy := defs.UnitConfigs[worker]
	spy.Vision.Space, spy.Vision.Adjacent = defs.VisibilitySpy, defs.VisibilitySpy
	defs.UnitConfigs[15] = spy
	attacker := defs.UnitConfigs[grunt]
	attacker.Attack_range = defs.RisqRange_ADJACENT
	attacker.Vision.Adjacent = defs.VisibilityGood
	defs.UnitConfigs[grunt] = attacker
	return harness.NewGame(t, "custom:hidden-target", 1, 3)
}

func spyGarrisonGame(t *testing.T) *harness.Game {
	t.Helper()
	t.Cleanup(func() { pristine.Restore(baseDir) })
	spaces := spaceWith(0, 0, flatTerrain, unit(15, 0)) + `,{"x":1,"y":0,"terrain":1,"zones":[{"x":0,"y":0,"building":{"id":201,"player":1},"units":[{"id":11,"player":1,"count":1}]}]}`
	writeCustomMap("spy-garrison", mapDoc("", spaces))
	spy := defs.UnitConfigs[worker]
	spy.Vision.Space, spy.Vision.Adjacent = defs.VisibilitySpy, defs.VisibilitySpy
	defs.UnitConfigs[15] = spy
	return harness.NewGame(t, "custom:spy-garrison", 1, 2)
}
