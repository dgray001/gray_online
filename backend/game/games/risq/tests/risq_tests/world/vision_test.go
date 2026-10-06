package world

import (
	"github.com/dgray001/gray_online/game/games/risq/internal/defs"
	"github.com/dgray001/gray_online/game/games/risq/tests/harness"
	"testing"
)

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
