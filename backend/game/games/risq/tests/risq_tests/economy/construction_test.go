package economy

import (
	"fmt"
	"testing"

	"github.com/dgray001/gray_online/game/games/risq/internal/defs"
	"github.com/dgray001/gray_online/game/games/risq/tests/harness"
	"github.com/dgray001/gray_online/game/games/risq/tests/harness/fakeboard"
)

func TestConstructionSpendsOnceAndCarriesProgress(t *testing.T) {
	g := buildGame(t, 1)
	p := g.Human(0)
	wood := g.Self(p).Resources.Wood
	g.Submit(p, housingOrders(g, p, 0)...)
	g.EndTurn()
	buildings := g.Self(p).Buildings
	if len(buildings) != 1 {
		t.Fatalf("buildings %+v, want one housing", buildings)
	}
	b := buildings[0]
	if b.BuildingID != 2 || !b.UnderConstruction || b.StaminaRemaining != 6 {
		t.Errorf("first-turn housing %+v, want under construction with 6 stamina remaining", b)
	}
	if got := g.Self(p).Resources.Wood; got != wood-30 {
		t.Errorf("first-turn wood %v, want %v", got, wood-30)
	}
	g.EndTurn()
	buildings = g.Self(p).Buildings
	if len(buildings) != 1 || buildings[0].InternalID != b.InternalID || buildings[0].UnderConstruction || buildings[0].StaminaRemaining != 0 {
		t.Errorf("second-turn buildings %+v, want the same completed housing", buildings)
	}
	g.EndTurn()
	if got := g.Self(p).Resources.Wood; got != wood-30 {
		t.Errorf("wood after completion %v, want a single charge: %v", got, wood-30)
	}
}

func buildGame(t *testing.T, workers int) *harness.Game {
	t.Helper()
	doc := fmt.Sprintf(`{"board_size":3,"players":2,"starting_bank":{"wood":120},"spaces":[{"x":0,"y":0,"terrain":1,"zones":[{"x":0,"y":0,"units":[{"id":1,"player":0,"count":%d},{"id":11,"player":0,"count":1}]}]},{"x":1,"y":0,"terrain":1,"zones":[{"x":0,"y":0,"units":[{"id":11,"player":0,"count":1}]}]},{"x":2,"y":0,"terrain":1},{"x":-3,"y":0,"terrain":1,"zones":[{"x":0,"y":0,"units":[{"id":1,"player":1,"count":1}]}]}]}`, workers)
	fakeboard.UseConfig(t, testConfig, nil, map[string]string{"build": doc})
	return startGame(t, "custom:build", 1, 2)
}

func housingOrders(g *harness.Game, human int, x int) []defs.OrderFromFrontend {
	var orders []defs.OrderFromFrontend
	for _, u := range g.Self(human).Units {
		if u.UnitID == 1 {
			orders = append(orders, harness.OrderBuild([]uint64{u.InternalID}, 2, x, 0, 0, 0))
		}
	}
	return orders
}

func TestConstructionAssistSharesBuildingAndCost(t *testing.T) {
	for _, workers := range []int{1, 2} {
		g := buildGame(t, workers)
		p := g.Human(0)
		wood := g.Self(p).Resources.Wood
		g.Submit(p, housingOrders(g, p, 0)...)
		g.EndTurn()
		buildings := g.Self(p).Buildings
		if len(buildings) != 1 {
			t.Fatalf("%d builders created %d buildings, want one", workers, len(buildings))
		}
		if got := g.Self(p).Resources.Wood; got != wood-30 {
			t.Errorf("%d builders left %v wood, want one charge: %v", workers, got, wood-30)
		}
		if buildings[0].BuildingID != 2 || buildings[0].UnderConstruction != (workers == 1) {
			t.Errorf("%d builders: housing %+v, want completed only with two builders", workers, buildings[0])
		}
		if len(g.Self(p).PlannedFoundations) != 0 {
			t.Error("materialized housing retained a planned foundation")
		}
		g.EndTurn()
		if after := g.Self(p); len(after.Buildings) != 1 || after.Buildings[0].InternalID != buildings[0].InternalID || after.Resources.Wood != wood-30 {
			t.Errorf("%d builders: next-turn state has duplicate housing or another charge", workers)
		}
	}
}

func TestLaterBuilderAssistsExistingConstruction(t *testing.T) {
	g := buildGame(t, 2)
	p := g.Human(0)
	workers := housingOrders(g, p, 0)
	g.Submit(p, workers[0])
	g.EndTurn()
	b := centerHousing(g, p)
	if !b.UnderConstruction || b.StaminaRemaining != 6 {
		t.Fatalf("setup did not leave unfinished housing: %+v", b)
	}
	order := g.Self(p).ActiveOrders[0].InternalID
	g.Submit(p, harness.Order(defs.OrderType_CancelOrder, nil, int64(order), false), workers[1])
	g.EndTurn()
	after := centerHousing(g, p)
	if after.InternalID != b.InternalID || after.UnderConstruction || g.Self(p).Resources.Wood != 90 || len(g.Self(p).Refusals()) != 0 {
		t.Errorf("later assistance did not finish the same housing with one payment: %+v", g.Self(p))
	}
}

func TestTerrainChangesConstructionSpeed(t *testing.T) {
	for name, c := range map[string]struct{ terrain, remaining, turns int }{"grass": {1, 6, 2}, "hills": {51, 8, 3}} {
		t.Run(name, func(t *testing.T) {
			spaces := fmt.Sprintf(`{"x":0,"y":0,"terrain":%d,"zones":[{"x":0,"y":0,"units":[{"id":1,"player":0,"count":1}]}]},{"x":1,"y":0,"terrain":1,"zones":[{"x":0,"y":0,"building":{"id":2,"player":0}}]},{"x":-2,"y":0,"terrain":1,"zones":[{"x":0,"y":0,"units":[{"id":1,"player":1,"count":1}]}]}`, c.terrain)
			g := scenarioGame(t, spaces)
			p := g.Human(0)
			g.Submit(p, housingOrders(g, p, 0)...)
			g.EndTurn()
			b := centerHousing(g, p)
			if !b.UnderConstruction || b.StaminaRemaining != c.remaining {
				t.Errorf("first-turn housing %+v, want %d stamina remaining", b, c.remaining)
			}
			turns := 1
			for centerHousing(g, p).UnderConstruction && turns < 6 {
				g.EndTurn()
				turns++
			}
			finished := centerHousing(g, p)
			if turns != c.turns || finished.UnderConstruction || finished.InternalID != b.InternalID {
				t.Errorf("housing completed after %d turns: %+v, want %d turns and same building", turns, finished, c.turns)
			}
			if got := g.Self(p).Resources.Wood; got != 90 {
				t.Errorf("wood %v, want one 30-wood payment", got)
			}
		})
	}
}
