package invariants

import (
	"fmt"
	"strings"
	"testing"

	"github.com/dgray001/gray_online/game/games/risq/internal/defs"
	"github.com/dgray001/gray_online/game/games/risq/tests/harness"
)

func economyScenario() scenario {
	return scenario{players: 3, turns: 7, orders: economyOrders, verify: verifyEconomy,
		setup: func(t *testing.T) *harness.Game {
			return invariantGame(t, 3, homeSpaces(3, 3)+centralSpace, testRegions, noConfig)
		},
	}
}

var homes = [][2]int{{-1, 0}, {1, 0}, {0, 1}}

func homeSpaces(players, workers int) string {
	var spaces []string
	for slot, home := range homes[:players] {
		spaces = append(spaces, fmt.Sprintf(`{"x":%d,"y":%d,"terrain":1,"zones":[{"x":0,"y":0,"building":{"id":201,"player":%d},"units":[{"id":11,"player":%d,"count":%d}]},{"x":1,"y":0,"resource":11}]}`, home[0], home[1], slot, slot, workers))
	}
	return strings.Join(spaces, ",")
}

const centralSpace = `,{"x":0,"y":0,"terrain":1}`
const testRegions = `,"regions":[{"name":"West","gold_bonus":7,"spaces":[[-1,0],[0,0]]},{"name":"East","gold_bonus":3,"spaces":[[1,0]]}]`

func pairedCenters(workers int) string {
	spaces := strings.Replace(homeSpaces(2, workers), `"resource":11`, `"building":{"id":201,"player":0}`, 1)
	return strings.Replace(spaces, `"resource":11`, `"building":{"id":201,"player":1}`, 1) + centralSpace
}

func economyOrders(g *harness.Game, turn, slot int) []defs.OrderFromFrontend {
	ids, home := unitIDs(g, slot, 11), homes[slot]
	center := buildingID(g, slot, 201)
	switch turn {
	case 0:
		return []defs.OrderFromFrontend{
			harness.OrderGather(ids[:1], home[0], home[1], 1, 0),
			harness.OrderBuild(ids[1:2], 202, home[0], home[1], 0, -1),
			harness.OrderGarrison(ids[2:3], center), harness.OrderProduce([]uint64{center}, 11),
		}
	case 1:
		return []defs.OrderFromFrontend{harness.OrderProduce([]uint64{center}, 11)}
	case 2:
		return []defs.OrderFromFrontend{harness.Order(defs.OrderType_UnitUngarrison, ids[2:3], 0, false), harness.Order(defs.OrderType_BuildingResearch, []uint64{center}, 301, false)}
	case 3:
		return []defs.OrderFromFrontend{harness.Order(defs.OrderType_UnitDelete, ids[len(ids)-1:], 0, false)}
	case 4:
		return []defs.OrderFromFrontend{harness.Order(defs.OrderType_BuildingDelete, []uint64{buildingID(g, slot, 202)}, 0, false)}
	case 5:
		return []defs.OrderFromFrontend{harness.Order(defs.OrderType_UnitMoveSpace, ids, harness.SpaceKey(0, 0), true)}
	}
	return nil
}

func verifyEconomy(g *harness.Game, turn int) {
	g.T.Helper()
	for slot := range 3 {
		state := g.Self(g.Human(slot))
		if failures := state.Refusals(); len(failures) > 0 {
			g.T.Fatalf("economy turn %d refusals: %v", turn, failures)
		}
		if turn == 0 && g.State(g.Human(slot)).Unit(unitIDs(g, slot, 11)[2]).GarrisonedIn == nil {
			g.T.Fatal("garrison never exercised")
		}
		if turn == 1 && state.PopulationLimit != 10 {
			g.T.Fatal("shelter did not complete")
		}
		if turn == 2 && g.State(g.Human(slot)).Unit(unitIDs(g, slot, 11)[2]).GarrisonedIn != nil {
			g.T.Fatal("ungarrison never exercised")
		}
		if turn == 6 && (len(state.Units) != 4 || len(state.Buildings) != 1 || !state.ResearchedTechs[301] || state.Resources.Wood <= 420) {
			g.T.Fatalf("economy incomplete: units %d, buildings %d, techs %v, wood %v", len(state.Units), len(state.Buildings), state.ResearchedTechs, state.Resources.Wood)
		}
	}
}

const sharedWorkers = `,{"x":0,"y":0,"terrain":1,"zones":[{"x":0,"y":0,"resource":1,"units":[{"id":11,"player":0,"count":1},{"id":11,"player":1,"count":1}]}]}`

func gatheringScenario() scenario {
	return scenario{players: 2, turns: 3, orders: gatheringOrders, verify: verifyGathering,
		setup: func(t *testing.T) *harness.Game {
			return invariantGame(t, 2, homeSpaces(2, 1)+sharedWorkers, testRegions, func() {
				config := defs.ResourceConfigs[1]
				config.Starting_resources = 3.5
				defs.ResourceConfigs[1] = config
			})
		},
	}
}

func gatheringOrders(g *harness.Game, turn, slot int) []defs.OrderFromFrontend {
	if turn != 0 {
		return nil
	}
	ids := unitIDs(g, slot, 11)
	return []defs.OrderFromFrontend{harness.OrderGather(ids[1:], 0, 0, 0, 0)}
}

func verifyGathering(g *harness.Game, turn int) {
	g.T.Helper()
	a, b := g.Self(g.Human(0)), g.Self(g.Human(1))
	if a.Resources.Food+b.Resources.Food != 1003.5 || a.Resources.Food != b.Resources.Food {
		g.T.Fatalf("contested gathering banks %v/%v, want equal shares totaling 1003.5", a.Resources.Food, b.Resources.Food)
	}
	if len(g.State(g.Human(0)).Space(0, 0).Resources) != 0 {
		g.T.Fatal("exhausted node survived")
	}
}

func constructionScenario() scenario {
	return scenario{players: 2, turns: 3, verify: verifyConstruction,
		setup: func(t *testing.T) *harness.Game {
			return invariantGame(t, 2, homeSpaces(2, 1)+strings.ReplaceAll(sharedWorkers, `"resource":1,`, ""), testRegions, noConfig)
		},
		orders: func(g *harness.Game, turn, slot int) []defs.OrderFromFrontend {
			if turn != 0 {
				return nil
			}
			ids := unitIDs(g, slot, 11)
			return []defs.OrderFromFrontend{harness.OrderBuild(ids[1:], 202, 0, 0, 0, 0)}
		},
	}
}

func verifyConstruction(g *harness.Game, turn int) {
	g.T.Helper()
	space := g.State(g.Human(0)).Space(0, 0)
	if len(space.Buildings) != 1 {
		g.T.Fatalf("contested foundation produced %d buildings", len(space.Buildings))
	}
	building := space.Buildings[0]
	if building.BuildingID != 202 || (turn >= 1 && building.UnderConstruction) {
		g.T.Fatal("winning shelter did not complete")
	}
	for slot := range 2 {
		state := g.Self(g.Human(slot))
		want := 500.0
		if slot == building.PlayerID {
			want -= 30
		}
		if state.Resources.Wood != want || len(state.PlannedFoundations) != 0 {
			g.T.Errorf("player %d foundation refund/cleanup: wood %v, foundations %v", slot, state.Resources.Wood, state.PlannedFoundations)
		}
	}
}

func orchardScenario() scenario {
	return scenario{players: 2, turns: 4, orders: orchardOrders, verify: verifyOrchard,
		setup: func(t *testing.T) *harness.Game {
			spaces := strings.Replace(homeSpaces(2, 1), `"resource":11`, `"building":{"id":203,"player":0}`, 1)
			spaces = strings.Replace(spaces, `"resource":11`, `"building":{"id":203,"player":1}`, 1)
			return invariantGame(t, 2, spaces+centralSpace, "", func() {
				config := defs.BuildingConfigs[203]
				config.Gather.Starting_resources = 2.5
				defs.BuildingConfigs[203] = config
			})
		},
	}
}

func orchardOrders(g *harness.Game, turn, slot int) []defs.OrderFromFrontend {
	if turn%2 == 1 {
		return []defs.OrderFromFrontend{harness.Order(defs.OrderType_UnitRenew, unitIDs(g, slot, 11), int64(buildingID(g, slot, 203)), false)}
	}
	home := homes[slot]
	return []defs.OrderFromFrontend{harness.OrderGather(unitIDs(g, slot, 11), home[0], home[1], 1, 0)}
}

func verifyOrchard(g *harness.Game, turn int) {
	g.T.Helper()
	for slot := range 2 {
		state := g.Self(g.Human(slot))
		if len(state.Refusals()) != 0 || state.Resources.Food != 500+2.5*float64(turn/2+1) || state.Resources.Wood != 500-40*float64((turn+1)/2) {
			g.T.Fatalf("orchard cycle %d bank %+v, failures %v", turn, state.Resources, state.Refusals())
		}
	}
}

func productionScenario() scenario {
	return scenario{players: 2, turns: 4, orders: productionOrders, verify: verifyProduction,
		setup: func(t *testing.T) *harness.Game {
			return invariantGame(t, 2, pairedCenters(5), "", func() {
				config := defs.BuildingConfigs[201]
				config.Population_support = 3
				defs.BuildingConfigs[201] = config
			})
		},
	}
}

func productionOrders(g *harness.Game, turn, slot int) []defs.OrderFromFrontend {
	if turn == 2 {
		return []defs.OrderFromFrontend{harness.Order(defs.OrderType_UnitDelete, unitIDs(g, slot, 11)[:1], 0, false)}
	}
	if turn != 0 {
		return nil
	}
	var ids []uint64
	for _, building := range g.Self(g.Human(slot)).Buildings {
		ids = append(ids, building.InternalID)
	}
	return []defs.OrderFromFrontend{harness.OrderProduce(ids, 11)}
}

func verifyProduction(g *harness.Game, turn int) {
	g.T.Helper()
	for slot := range 2 {
		state := g.Self(g.Human(slot))
		queued := 0
		for _, building := range state.Buildings {
			queued += len(building.ProductionQueue)
		}
		if state.Resources.Food != 400 {
			g.T.Fatal("production charged more than its two reservations")
		}
		if turn <= 1 && (len(state.Units) != 6 || queued != 1) {
			g.T.Fatalf("last-slot contest: population %d, queued %d", len(state.Units), queued)
		}
		if turn == 3 && (len(state.Units) != 6 || queued != 0) {
			g.T.Fatal("blocked production did not resume after deletion")
		}
	}
}

func budgetScenario() scenario {
	return scenario{players: 2, turns: 3, orders: productionOrders, verify: verifyBudget,
		setup: func(t *testing.T) *harness.Game {
			return invariantGame(t, 2, pairedCenters(1), "", func() {
				config := defs.UnitConfigs[11]
				config.Cost = defs.RisqResourceCost{Food: 500, Wood: 500, Stone: 500, Gold: 500}
				defs.UnitConfigs[11] = config
			})
		},
	}
}

func verifyBudget(g *harness.Game, turn int) {
	g.T.Helper()
	for slot := range 2 {
		state := g.Self(g.Human(slot))
		if state.Resources.Food != 0 || state.Resources.Wood != 0 || state.Resources.Stone != 0 {
			g.T.Fatalf("budget was not spent exactly once: %+v", state.Resources)
		}
		want := 2
		if turn == 2 {
			want = 1
		}
		if len(state.Units) != want {
			g.T.Fatalf("budget allowed %d units, want %d", len(state.Units), want)
		}
		if turn == 0 && len(state.Refusals()) != 1 {
			g.T.Fatal("insufficient-budget production was not refused")
		}
	}
}
