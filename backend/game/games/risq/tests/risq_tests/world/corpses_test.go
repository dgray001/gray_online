package world

import (
	"maps"
	"slices"
	"testing"

	"github.com/dgray001/gray_online/game/games/risq/internal/defs"
	"github.com/dgray001/gray_online/game/games/risq/tests/harness"
	"github.com/dgray001/gray_online/game/games/risq/tests/harness/fakeboard"
	"github.com/gin-gonic/gin"
)

func corpseGame(t *testing.T, visibility defs.VisibilityLevel) *harness.Game {
	t.Helper()
	spaces := spaceWith(0, 0, grass, unit(villager, 0)+","+unit(villager, 0)+","+unit(infantry, 1)) + "," +
		spaceWith(-3, 0, grass, unit(villager, 0)) + "," + spaceWith(3, 0, grass, unit(villager, 1))
	fakeboard.UseConfig(t, testConfig, nil, map[string]string{"world": mapDoc("", spaces)})
	config := defs.UnitConfigs[infantry]
	config.Attack_blunt = 200
	config.Vision.Space = visibility
	config.Vision.Adjacent, config.Vision.Secondary = defs.VisibilityUnexplored, defs.VisibilityUnexplored
	defs.UnitConfigs[infantry] = config
	g := harness.NewGame(t, "custom:world", 1, 2)
	attacker := unitAt(g, g.Human(1), infantry, 0, 0)
	g.Action(g.Human(1), "set-unit-behavior", gin.H{"internal_ids": []uint64{attacker}, "stance": uint8(defs.UnitStance_PASSIVE), "attack_back": false})
	return g
}

func killZoneUnits(g *harness.Game) {
	g.T.Helper()
	attacker := unitAt(g, g.Human(1), infantry, 0, 0)
	g.Submit(g.Human(1), harness.Order(defs.OrderType_UnitAttackZone, []uint64{attacker}, harness.ZoneKey(0, 0, 0, 0), false))
	g.EndTurn()
}

func assertCorpses(g *harness.Game, want map[uint64]harness.CorpseState) {
	g.T.Helper()
	corpses := g.State(g.Human(1)).Space(0, 0).Zones[1][1].Corpses
	if len(corpses) != len(want) {
		g.T.Fatalf("corpses %+v, want %+v", corpses, want)
	}
	for i, corpse := range corpses {
		if corpse != want[corpse.InternalID] {
			g.T.Fatalf("corpse %+v, want %+v", corpse, want[corpse.InternalID])
		}
		if i > 0 && corpses[i-1].InternalID >= corpse.InternalID {
			g.T.Fatal("corpses are not sorted by internal ID")
		}
	}
}

func expectedCorpses(g *harness.Game) map[uint64]harness.CorpseState {
	g.T.Helper()
	want := make(map[uint64]harness.CorpseState)
	for _, unit := range g.Self(g.Human(0)).Units {
		if unit.Space == (harness.Coord{}) {
			want[unit.InternalID] = harness.CorpseState{InternalID: unit.InternalID, UnitID: unit.UnitID, PlayerID: unit.PlayerID, Turns: 1}
		}
	}
	return want
}

func TestCorpseLifetime(t *testing.T) {
	g := corpseGame(t, defs.VisibilityGood)
	want := expectedCorpses(g)
	killZoneUnits(g)
	assertCorpses(g, want)
	for id := range want {
		if g.State(g.Human(0)).Unit(id) != nil {
			t.Fatal("corpse retained a live unit")
		}
	}
	for age := uint8(2); age <= 3; age++ {
		g.EndTurn()
		for id, corpse := range want {
			corpse.Turns = age
			want[id] = corpse
		}
		if age == 3 {
			clear(want)
		}
		assertCorpses(g, want)
	}
}

func TestCorpseAgesIndependently(t *testing.T) {
	g := corpseGame(t, defs.VisibilityGood)
	victims := expectedCorpses(g)
	ids := slices.Sorted(maps.Keys(victims))
	want := make(map[uint64]harness.CorpseState)
	attacker := unitAt(g, g.Human(1), infantry, 0, 0)
	for _, id := range ids {
		g.Submit(g.Human(1), harness.OrderAttackUnit([]uint64{attacker}, id))
		g.EndTurn()
		for previous, corpse := range want {
			corpse.Turns++
			want[previous] = corpse
		}
		want[id] = victims[id]
		assertCorpses(g, want)
	}
	g.EndTurn()
	delete(want, ids[0])
	corpse := want[ids[1]]
	corpse.Turns = 2
	want[ids[1]] = corpse
	assertCorpses(g, want)
	g.EndTurn()
	clear(want)
	assertCorpses(g, want)
}

func TestCorpseVisibility(t *testing.T) {
	for name, visibility := range map[string]defs.VisibilityLevel{"fog": defs.VisibilityFog, "poor": defs.VisibilityPoor, "good": defs.VisibilityGood, "spy": defs.VisibilitySpy} {
		t.Run(name, func(t *testing.T) {
			g := corpseGame(t, visibility)
			killZoneUnits(g)
			space := g.State(g.Human(1)).Space(0, 0)
			corpses := space.Zones[1][1].Corpses
			want := 0
			if visibility >= defs.VisibilityGood {
				want = 2
			}
			if space.Visibility != int(visibility) || len(corpses) != want || (visibility < defs.VisibilityGood && corpses != nil) {
				t.Fatalf("visibility %d corpses %+v, want visibility %d and %d corpses", space.Visibility, corpses, visibility, want)
			}
		})
	}
}
