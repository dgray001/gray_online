package invariants

import (
	"strings"
	"testing"

	"github.com/dgray001/gray_online/game/games/risq/internal/defs"
	"github.com/dgray001/gray_online/game/games/risq/tests/harness"
	"github.com/gin-gonic/gin"
)

const fighters = `,{"x":0,"y":0,"terrain":1,"zones":[{"x":0,"y":0,"units":[{"id":13,"player":0,"count":1},{"id":13,"player":1,"count":1}]}]}`

func combatScenario() scenario {
	return scenario{players: 2, turns: 4, orders: combatOrders, verify: verifyCombat,
		setup: func(t *testing.T) *harness.Game {
			return invariantGame(t, 2, homeSpaces(2, 1)+fighters, testRegions, func() {
				config := defs.UnitConfigs[13]
				config.Max_health, config.Turn_stamina = 8, 3
				config.Attack_type, config.Attack_blunt = defs.AttackType_BLUNT, 10
				config.Defense_blunt, config.Penetration_blunt = 0, 0
				defs.UnitConfigs[13] = config
			})
		},
	}
}

func combatOrders(g *harness.Game, turn, slot int) []defs.OrderFromFrontend {
	if turn != 0 {
		return nil
	}
	return []defs.OrderFromFrontend{harness.OrderAttackUnit(unitIDs(g, slot, 13), unitIDs(g, 1-slot, 13)[0])}
}

func verifyCombat(g *harness.Game, turn int) {
	g.T.Helper()
	for slot := range 2 {
		units := unitIDs(g, slot, 13)
		if turn == 0 && (len(units) != 1 || g.State(g.Human(slot)).Unit(units[0]).CombatStats.Health >= 8) {
			g.T.Fatal("combat did not damage both fighters")
		}
		if turn == 3 && len(units) != 0 {
			g.T.Fatalf("mutual combat left fighters: %+v", g.Self(g.Human(slot)).Units)
		}
	}
}

func evictionScenario() scenario {
	return scenario{players: 2, turns: 5, capLossTurn: 4, orders: evictionOrders, verify: verifyEviction,
		setup: func(t *testing.T) *harness.Game {
			return invariantGame(t, 2, homeSpaces(2, 3)+centralSpace, testRegions, func() {
				config := defs.BuildingConfigs[1]
				config.Garrison_capacity = 1
				defs.BuildingConfigs[1] = config
			})
		},
	}
}

func evictionOrders(g *harness.Game, turn, slot int) []defs.OrderFromFrontend {
	if turn == 0 || turn == 2 {
		return []defs.OrderFromFrontend{harness.OrderGarrison(unitIDs(g, slot, 1), buildingID(g, slot, 1))}
	}
	if turn == 1 {
		for _, unit := range g.Self(g.Human(slot)).Units {
			if unit.GarrisonedIn != nil {
				return []defs.OrderFromFrontend{harness.Order(defs.OrderType_UnitDelete, []uint64{unit.InternalID}, 0, false)}
			}
		}
		g.T.Fatal("garrison capacity contest admitted no units")
	}
	if turn == 3 {
		return []defs.OrderFromFrontend{harness.Order(defs.OrderType_BuildingDelete, []uint64{buildingID(g, slot, 1)}, 0, false)}
	}
	return nil
}

func verifyEviction(g *harness.Game, turn int) {
	g.T.Helper()
	for slot := range 2 {
		state := g.Self(g.Human(slot))
		garrisoned := 0
		for _, unit := range state.Units {
			if unit.GarrisonedIn != nil {
				garrisoned++
			}
		}
		if (turn == 0 || turn == 2) && garrisoned != 1 {
			g.T.Fatalf("garrison admitted %d units, want 1", garrisoned)
		}
		if turn >= 1 && len(state.Units) != 2 {
			g.T.Fatal("deleting a garrisoned unit left a ghost or removed another unit")
		}
		if turn >= 3 && (garrisoned != 0 || len(state.Buildings) != 0 || state.PopulationLimit != 0) {
			g.T.Fatal("building deletion did not release garrison and population support")
		}
	}
}

func captureScenario() scenario {
	return scenario{players: 2, turns: 4, capLossTurn: 3, orders: captureOrders, verify: verifyCapture,
		setup: func(t *testing.T) *harness.Game {
			spaces := strings.Replace(homeSpaces(2, 1), `"units":[{"id":1,`, `"units":[{"id":11,`, 1)
			g := invariantGame(t, 2, spaces+centralSpace, testRegions, noConfig)
			g.Action(g.Human(0), "set-unit-behavior", gin.H{"internal_ids": unitIDs(g, 0, 11), "stance": 1, "attack_back": false})
			return g
		},
	}
}

func captureOrders(g *harness.Game, turn, slot int) []defs.OrderFromFrontend {
	if slot == 1 && turn == 2 {
		return []defs.OrderFromFrontend{harness.Order(defs.OrderType_BuildingDelete, []uint64{buildingID(g, 1, 1)}, 0, false)}
	}
	if slot == 0 && turn != 2 {
		x := 0
		if turn == 1 {
			x = 1
		}
		return []defs.OrderFromFrontend{harness.OrderMove(unitIDs(g, 0, 11), x, 0)}
	}
	return nil
}

func verifyCapture(g *harness.Game, turn int) {
	g.T.Helper()
	x, want := 1, []int{0, 1, 0, -1}[turn]
	if turn == 0 {
		x = 0
	}
	owner := g.State(g.Human(0)).Space(x, 0).Ownership
	if owner == nil || *owner != want {
		g.T.Fatalf("capture turn %d space (%d,0): owner %v, want %d", turn, x, owner, want)
	}
	if turn == 1 && g.State(g.Human(0)).Unit(unitIDs(g, 0, 11)[0]).Space.X != 1 {
		g.T.Fatal("soldier did not enter enemy building's space")
	}
}

