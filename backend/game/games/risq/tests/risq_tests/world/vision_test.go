package world

import (
	"github.com/dgray001/gray_online/game"
	"github.com/dgray001/gray_online/game/games/risq/internal/defs"
	"github.com/dgray001/gray_online/game/games/risq/tests/harness"
	"github.com/dgray001/gray_online/game/games/risq/tests/harness/fakeboard"
	"github.com/gin-gonic/gin"
	"reflect"
	"testing"
)

func TestTickHistorySpyVisibility(t *testing.T) {
	t.Run("garrisoned-and-deleted-actor", func(t *testing.T) {
		g := spyGarrisonGame(t)
		observer, owner := g.Human(0), g.Human(1)
		id, building := firstUnit(g, owner).InternalID, g.Self(owner).Buildings[0].InternalID
		g.Submit(owner, harness.OrderGarrison([]uint64{id}, building))
		g.EndTurn()
		if g.State(owner).Unit(id).GarrisonedIn == nil || g.State(observer).Space(1, 0).Visibility != 4 {
			t.Fatal("fixture did not garrison under spy coverage")
		}
		archived := harness.RequireReplay(t, g.State(observer)).Unit(id)
		if archived == nil {
			t.Fatal("spy cannot inspect garrisoned actor history")
		}
		harness.RequireTickAction(t, archived.TickActions, "garrison")
		g.Submit(owner, harness.Order(defs.OrderType_UnitDelete, []uint64{id}, 0, false))
		g.EndTurn()
		if g.State(owner).Unit(id) != nil {
			t.Fatal("fixture did not delete garrisoned actor")
		}
		archived = harness.RequireReplay(t, g.State(observer)).Unit(id)
		if archived == nil {
			t.Fatal("spy lost deleted actor history")
		}
		harness.RequireTickAction(t, archived.TickActions, "delete")
	})
	t.Run("hidden-target", func(t *testing.T) {
		g := spyHiddenTargetGame(t)
		observer, owner, enemy := g.Human(0), g.Human(1), g.Human(2)
		attacker, target := firstUnit(g, owner), firstUnit(g, enemy)
		g.Action(enemy, "set-unit-behavior", gin.H{"internal_ids": []uint64{target.InternalID}, "stance": uint8(defs.UnitStance_PASSIVE), "attack_back": false})
		if g.State(observer).Space(2, 0).Visibility >= good || g.State(observer).Unit(attacker.InternalID) == nil {
			t.Fatal("fixture did not separate actor and target visibility")
		}
		g.Submit(owner, harness.OrderAttackUnit([]uint64{attacker.InternalID}, target.InternalID))
		g.EndTurn()
		if firstUnit(g, enemy).CombatStats.Health >= target.CombatStats.Health {
			t.Fatal("fixture did not attack the hidden target")
		}
		actions := g.State(observer).Unit(attacker.InternalID).TickActions
		harness.RequireTickAction(t, actions, "attack")
		for _, a := range actions {
			if a.Intent.TargetLocation.Space == target.Space || a.Execute.TargetLocation.Space == target.Space {
				t.Errorf("spy leaked hidden target location: %+v", a)
			}
		}
		replay := harness.RequireReplay(t, g.State(observer))
		if replay.Unit(target.InternalID) != nil || replay.UnitAt(target.InternalID, replay.TickCount) != nil {
			t.Fatal("hidden target stats leaked into archive or frames")
		}
	})
	t.Run("hidden-travel", func(t *testing.T) {
		g := spyTravelGame(t)
		observer, owner := g.Human(0), g.Human(1)
		id := firstUnit(g, owner).InternalID
		if g.State(observer).Unit(id) != nil {
			t.Fatal("fixture traveller already visible")
		}
		g.Submit(owner, harness.OrderMove([]uint64{id}, 0, 0))
		g.EndTurn()
		own, seen := g.State(owner).Unit(id), g.State(observer).Unit(id)
		if own.Space != (harness.Coord{}) || seen == nil {
			t.Fatal("fixture traveller did not enter spy coverage")
		}
		if len(seen.TickActions) == 0 || len(seen.TickActions) >= len(own.TickActions) {
			t.Fatal("spy history does not preserve a hidden travel interval")
		}
		for _, a := range seen.TickActions {
			if a.Intent.Location.Space.X > 1 {
				t.Errorf("spy saw hidden movement: %+v", a)
			}
		}
		replay := harness.RequireReplay(t, g.State(observer))
		if replay.UnitAt(id, 0) != nil {
			t.Fatal("hidden traveller leaked into baseline")
		}
	})
	g := spyGame(t)
	observer, owner := g.Human(0), g.Human(1)
	id := firstUnit(g, owner).InternalID
	g.Submit(owner, harness.OrderGather([]uint64{id}, 1, 0, 0, 0))
	g.EndTurn()
	own, seen := g.State(owner).Unit(id), g.State(observer).Unit(id)
	if own == nil || seen == nil || g.State(observer).Space(1, 0).Visibility != 4 {
		t.Fatal("fixture did not expose the enemy through spy vision")
	}
	if len(own.TickActions) == 0 || !reflect.DeepEqual(own.TickActions, seen.TickActions) {
		t.Fatalf("spy history differs from owner history: own %+v, seen %+v", own.TickActions, seen.TickActions)
	}
	ordinary := visibilityGame(t, float64(defs.VisibilityMode_ALL_VISIBLE))
	observer, owner = ordinary.Human(0), ordinary.Human(1)
	id = firstUnit(ordinary, owner).InternalID
	ordinary.Submit(owner, harness.OrderGather([]uint64{id}, 4, 0, 1, 0))
	ordinary.EndTurn()
	if len(ordinary.State(owner).Unit(id).TickActions) == 0 {
		t.Fatal("owner history missing under ordinary vision")
	}
	if enemy := ordinary.State(observer).Unit(id); enemy == nil || len(enemy.TickActions) != 0 {
		t.Fatalf("ordinary visibility leaked history: %+v", enemy)
	}
}

func visibilityGame(t *testing.T, setting any) *harness.Game {
	t.Helper()
	spaces := spaceWith(0, 0, grass, unit(villager, 0)) + `,{"x":4,"y":0,"terrain":1,"zones":[` +
		`{"x":0,"y":0,"units":[` + unit(villager, 1) + `]},` +
		`{"x":1,"y":0,"resource":21},{"x":0,"y":1,"building":{"id":2,"player":1}}]}`
	fakeboard.UseConfig(t, testConfig, nil, map[string]string{"visibility": mapDoc("", spaces)})
	g := harness.NewUnstartedGameWithSettings(t, "custom:visibility", 1, 2, map[string]any{"visibility": setting})
	game.Game_StartGame(g.Risq)
	return g
}

func TestVisibilityModesPersistAcrossTurns(t *testing.T) {
	for _, c := range []struct {
		setting any
		want    int
	}{{nil, unexplored}, {float64(0), unexplored}, {float64(1), unexplored}, {float64(2), fog}, {float64(3), good}, {float64(4), unexplored}, {float64(-1), unexplored}} {
		g := visibilityGame(t, c.setting)
		for turn := 0; turn < 3; turn++ {
			for slot := 0; slot < 2; slot++ {
				space := g.State(g.Human(slot)).Space(4*(1-slot), 0)
				if space.Visibility != c.want {
					t.Errorf("setting %v, turn %d, player %d: vision %d, want %d", c.setting, turn, slot, space.Visibility, c.want)
				}
			}
			g.EndTurn()
		}
	}
}

func TestExploredStartsWithFogCaches(t *testing.T) {
	g := visibilityGame(t, float64(defs.VisibilityMode_EXPLORED))
	west := g.Human(0)
	space := g.State(west).Space(4, 0)
	if space.TerrainID == nil || space.Ownership == nil || *space.Ownership != 1 || len(space.Resources) != 1 || len(space.Buildings) != 1 {
		t.Fatalf("explored space is missing initial terrain, ownership, resource or building caches: %+v", space)
	}
	if len(space.Units) != 0 {
		t.Fatal("explored space revealed enemy units")
	}
}

func TestAllVisibleShowsEnemyUnitsWithoutOrders(t *testing.T) {
	g := visibilityGame(t, float64(defs.VisibilityMode_ALL_VISIBLE))
	west, east := g.Human(0), g.Human(1)
	id := firstUnit(g, east).InternalID
	g.Submit(east, harness.OrderGather([]uint64{id}, 4, 0, 1, 0))
	g.EndTurn()
	if own := g.State(east).Unit(id); own == nil || len(own.ActiveOrders) == 0 {
		t.Fatal("enemy has no gathering order")
	}
	if enemy := g.State(west).Unit(id); enemy == nil || len(enemy.ActiveOrders) != 0 {
		t.Fatalf("good visibility should reveal the unit without its orders: %+v", enemy)
	}
}

const (
	unexplored = 0
	fog        = 1
	poor       = 2
	good       = 3
)

// A row of five spaces; P0's villager at the west end, P1's at the east end
func corridorGame(t *testing.T) (*harness.Game, int, int) {
	t.Helper()
	spaces := spaceWith(0, 0, grass, unit(villager, 0)) + "," + spaceWith(1, 0, grass, "") + "," + spaceWith(2, 0, grass, "") + "," +
		spaceWith(3, 0, grass, "") + "," + spaceWith(4, 0, grass, unit(villager, 1))
	g := startGame(t, mapDoc("", spaces))
	return g, g.Human(0), g.Human(1)
}

func TestVisionFallsOffWithDistance(t *testing.T) {
	g, west, _ := corridorGame(t)
	state := g.State(west)
	for x, want := range []int{good, poor, unexplored, unexplored, unexplored} {
		if got := state.Space(x, 0).Visibility; got != want {
			t.Errorf("space (%d,0) visibility %d, want %d", x, got, want)
		}
	}
}

func TestUnexploredSpacesHideTheirContents(t *testing.T) {
	g, west, _ := corridorGame(t)
	hidden := g.State(west).Space(4, 0)
	if hidden.TerrainID != nil || hidden.Ownership != nil || hidden.Zones != nil {
		t.Errorf("an unexplored space leaked terrain %v, owner %v or zones", hidden.TerrainID, hidden.Ownership)
	}
	if hidden.Units != nil || hidden.UnitCount != nil || hidden.Buildings != nil || hidden.Resources != nil {
		t.Errorf("the enemy's space leaked units %v, count %v, buildings or resources", hidden.Units, hidden.UnitCount)
	}
}

// Walks the enemy villager to a space, failing if it takes more than ten turns
func walkEnemyTo(t *testing.T, g *harness.Game, east int, enemy uint64, x, y int) {
	t.Helper()
	g.Submit(east, harness.OrderMove([]uint64{enemy}, x, y))
	for turn := 0; g.State(east).Unit(enemy).Space != (harness.Coord{X: x, Y: y}); turn++ {
		if turn == 10 {
			t.Fatalf("the enemy never reached (%d,%d)", x, y)
		}
		g.EndTurn()
	}
}

func TestEnemyUnitsAreListedOnlyUnderGoodVision(t *testing.T) {
	g, west, east := corridorGame(t)
	enemy := g.Self(east).Units[0].InternalID
	if g.State(west).Unit(enemy) != nil {
		t.Error("an enemy unit far beyond vision is listed")
	}
	walkEnemyTo(t, g, east, enemy, 1, 0)
	if space := g.State(west).Space(1, 0); space.Visibility != poor || g.State(west).Unit(enemy) != nil {
		t.Errorf("enemy at a poor-vision space: visibility %d, listed %t; want poor and not listed", space.Visibility, g.State(west).Unit(enemy) != nil)
	}
	walkEnemyTo(t, g, east, enemy, 0, 0)
	if g.State(west).Unit(enemy) == nil {
		t.Error("an enemy unit in P0's own good-vision space is not listed")
	}
}

func TestPoorVisionShowsOnlyAUnitCount(t *testing.T) {
	g, west, east := corridorGame(t)
	enemy := g.Self(east).Units[0]
	count := func() int {
		if n := g.State(west).Space(1, 0).UnitCount; n != nil {
			return *n
		}
		return -1
	}
	if count() != 0 {
		t.Fatalf("space (1,0) reports %d units before anyone arrives, want a count of 0", count())
	}
	g.Submit(east, harness.OrderMove([]uint64{enemy.InternalID}, 1, 0))
	for turn := 0; turn < 8 && count() != 1; turn++ {
		g.EndTurn()
	}
	space := g.State(west).Space(1, 0)
	if space.Visibility != poor || count() != 1 || space.Units != nil {
		t.Errorf("space (1,0): visibility %d, count %d, units listed %t; want poor vision and only a count of 1", space.Visibility, count(), space.Units != nil)
	}
	for _, row := range space.Zones {
		for _, zone := range row {
			if zone.Units != nil {
				t.Errorf("zone %v lists units %v under poor vision, want only counts", zone.Coordinate, zone.Units)
			}
		}
	}
}

func TestLeftBehindSpacesFallBackToFogButStayExplored(t *testing.T) {
	g, west, _ := corridorGame(t)
	if turns := moveUntilArrived(g, west, 2, 0, 6); turns < 1 {
		t.Fatal("the unit never reached (2,0)")
	}
	origin := g.State(west).Space(0, 0)
	if origin.Visibility != fog || origin.TerrainID == nil {
		t.Errorf("origin: visibility %d, terrain %v; want fog with its terrain remembered", origin.Visibility, origin.TerrainID)
	}
	state := g.State(west)
	if ahead, beyond := state.Space(3, 0).Visibility, state.Space(4, 0).Visibility; ahead != poor || beyond != unexplored {
		t.Errorf("(3,0) visibility %d and (4,0) visibility %d, want poor and unexplored", ahead, beyond)
	}
}

func TestSpyTierVision(t *testing.T) {
	g := spyGame(t)
	p0, p1 := g.Human(0), g.Human(1)

	villagerID := g.Self(p1).Units[0].InternalID
	g.Submit(p1, harness.OrderGather([]uint64{villagerID}, 1, 0, 0, 0))
	g.EndTurn()

	enemy := g.State(p0).Unit(villagerID)
	if g.State(p0).Space(1, 0).Visibility != 4 || enemy == nil || len(enemy.ActiveOrders) != 1 || enemy.ActiveOrders[0].OrderType != uint8(defs.OrderType_UnitGather) {
		t.Fatalf("spy vision did not reveal the enemy's gathering order: %+v", enemy)
	}

	orderID := enemy.ActiveOrders[0].InternalID
	spyID := g.Self(p0).Units[0].InternalID
	g.Submit(p0, harness.Order(defs.OrderType_UnitDelete, []uint64{spyID}, 0, false))
	g.EndTurn()

	ownerView := g.State(p1).Unit(villagerID)
	if ownerView == nil || len(ownerView.ActiveOrders) != 1 || ownerView.ActiveOrders[0].InternalID != orderID {
		t.Fatal("the enemy's gathering order did not remain active")
	}
	enemy = g.State(p0).Unit(villagerID)
	if g.State(p0).Space(1, 0).Visibility != 3 || enemy == nil || len(enemy.ActiveOrders) != 0 {
		t.Errorf("enemy orders were not hidden at good visibility: %+v", enemy)
	}
}
