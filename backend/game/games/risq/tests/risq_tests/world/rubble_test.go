package world

import (
	"testing"

	"github.com/dgray001/gray_online/game/games/risq/internal/defs"
	"github.com/dgray001/gray_online/game/games/risq/tests/harness"
	"github.com/dgray001/gray_online/game/games/risq/tests/harness/fakeboard"
)

func rubbleGame(t *testing.T, configure ...func()) *harness.Game {
	t.Helper()
	spaces := `{"x":0,"y":0,"terrain":1,"zones":[{"x":0,"y":0,"building":{"id":2,"player":1},"units":[{"id":13,"player":0,"count":10},{"id":1,"player":0,"count":1}]}]},{"x":-1,"y":0,"terrain":1},{"x":-2,"y":0,"terrain":1,"zones":[{"x":0,"y":0,"building":{"id":1,"player":0},"units":[{"id":1,"player":0,"count":1}]}]},{"x":2,"y":0,"terrain":1,"zones":[{"x":0,"y":0,"building":{"id":1,"player":1}}]}`
	fakeboard.UseConfig(t, testConfig, nil, map[string]string{"world": mapDoc("", spaces)})
	for _, apply := range configure {
		apply()
	}
	g := harness.NewGame(t, "custom:world", 1, 2)
	owner := g.Human(0)
	ids := make([]uint64, 0)
	for _, unit := range g.Self(owner).Units {
		if unit.UnitID == 13 {
			ids = append(ids, unit.InternalID)
		}
	}
	target := g.State(owner).Space(0, 0).Zones[1][1].Building.InternalID
	g.Submit(owner, harness.Order(defs.OrderType_UnitAttackBuilding, ids, int64(target), false))
	g.Submit(g.Human(1))
	return g
}

func TestRubbleClearsWhenOccupied(t *testing.T) {
	for name, builderX := range map[string]int{"building": 0, "foundation": -2} {
		t.Run(name, func(t *testing.T) {
			g := rubbleGame(t)
			owner := g.Human(0)
			g.Submit(owner, harness.OrderBuild([]uint64{unitAt(g, owner, villager, builderX, 0)}, housing, 0, 0, 0, 0))
			g.EndTurn()
			z := g.State(owner).Space(0, 0).Zones[1][1]
			if z.DestroyedBuilding != nil || z.DestroyedBuildingTurns != nil {
				t.Fatal("occupied zones must not retain rubble")
			}
			if (builderX == 0 && z.Building == nil) || (builderX != 0 && len(g.Self(owner).PlannedFoundations) != 1) {
				t.Fatalf("expected replacement %s was not created", name)
			}
		})
	}
}

func TestRubbleCacheStaysFrozenUntilRefreshed(t *testing.T) {
	g := rubbleGame(t, func() {
		for _, id := range []uint32{villager, 13} {
			config := defs.UnitConfigs[id]
			config.Turn_stamina = 100
			config.Vision.Secondary = defs.VisibilityUnexplored
			defs.UnitConfigs[id] = config
		}
	})
	owner := g.Human(0)
	ids := make([]uint64, 0)
	for _, unit := range g.Self(owner).Units {
		ids = append(ids, unit.InternalID)
	}
	g.Submit(owner, harness.OrderMove(ids, -2, 0))
	g.EndTurn()
	space := g.State(owner).Space(0, 0)
	remembered := space.Zones[1][1]
	if space.Visibility != fog || remembered.DestroyedBuilding == nil || remembered.DestroyedBuildingTurns == nil || *remembered.DestroyedBuilding != 2 || *remembered.DestroyedBuildingTurns != 2 {
		t.Fatalf("setup did not leave seen rubble in fog: %+v", remembered)
	}
	g.EndTurn()
	cached := g.State(owner).Space(0, 0).Zones[1][1]
	if cached.DestroyedBuilding == nil || cached.DestroyedBuildingTurns == nil || *cached.DestroyedBuilding != 2 || *cached.DestroyedBuildingTurns != 2 {
		t.Fatalf("fog lost last-seen rubble: %+v", cached)
	}
	g.Submit(owner, harness.OrderMove(ids, 0, 0))
	g.EndTurn()
	refreshed := g.State(owner).Space(0, 0).Zones[1][1]
	if refreshed.DestroyedBuilding != nil || refreshed.DestroyedBuildingTurns != nil {
		t.Fatalf("refreshed cache retained stale rubble: %+v", refreshed)
	}
}

func TestRubbleLifetime(t *testing.T) {
	g := rubbleGame(t)
	for age := uint8(1); age <= 3; age++ {
		z := g.State(g.Human(0)).Space(0, 0).Zones[1][1]
		if z.Building != nil {
			t.Fatal("target building was not razed")
		}
		if age <= 2 {
			if z.DestroyedBuilding == nil || *z.DestroyedBuilding != 2 || z.DestroyedBuildingTurns == nil || *z.DestroyedBuildingTurns != age {
				t.Fatalf("age %d: incorrect rubble snapshot: %+v", age, z)
			}
		} else if z.DestroyedBuilding != nil || z.DestroyedBuildingTurns != nil {
			t.Fatal("expired rubble fields must be omitted")
		}
		if age < 3 {
			g.EndTurn()
		}
	}
}
