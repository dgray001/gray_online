package economy

import (
	"fmt"
	"strings"
	"testing"

	"github.com/dgray001/gray_online/game/games/risq/internal/defs"
	"github.com/dgray001/gray_online/game/games/risq/tests/harness"
	"github.com/dgray001/gray_online/game/games/risq/tests/harness/fakeboard"
	"github.com/gin-gonic/gin"
)

func TestProductionQueuePreservesDistinctItemOrder(t *testing.T) {
	g := economyGame(t, `{"food":300,"wood":100}`, `{"x":0,"y":0,"terrain":1,"zones":[{"x":0,"y":0,"building":{"id":22,"player":0},"units":[{"id":1,"player":0,"count":1}]}]},`+remoteVillager, `,"unlimited_population":true`)
	p := g.Human(0)
	b := g.Self(p).Buildings[0].InternalID
	g.Action(p, "set-default-unit-behavior", gin.H{"stance": uint8(defs.UnitStance_AGGRESSIVE), "attack_back": false, "interrupt_current": true, "target_priority": []uint8{1, 0, 255}})
	g.Submit(p, harness.OrderProduce([]uint64{b}, 101), harness.OrderProduce([]uint64{b}, 102))
	for turn := 1; turn <= 4; turn++ {
		g.EndTurn()
		state := g.Self(p)
		for _, player := range g.Risq.ToFrontend(uint64(p+1), false)["players"].([]gin.H) {
			if player["player"].(gin.H)["player_id"] != g.PlayerID(p) {
				continue
			}
			for _, unit := range player["units"].([]gin.H) {
				want := defs.UnitStance_AGGRESSIVE
				if unit["unit_id"] == uint32(1) {
					want = defs.UnitStance_PASSIVE
				}
				if unit["stance"] != want {
					t.Fatalf("incorrect default stance: %+v", unit)
				}
				military := unit["unit_id"] != uint32(1)
				priority := unit["target_priority"].([]int)
				if unit["attack_back"] != !military || unit["interrupt_current"] != military || (military && (len(priority) != 1 || priority[0] != 1)) || (!military && len(priority) != 0) {
					t.Fatalf("incorrect default behavior: %+v", unit)
				}
			}
		}
		counts := map[uint32]int{}
		for _, u := range state.Units {
			counts[u.UnitID]++
		}
		wantPiercing := 0
		if turn >= 3 {
			wantPiercing = 1
		}
		if counts[1] != 1 || counts[101] != 1 || counts[102] != wantPiercing || state.Resources.Food != 180 || state.Resources.Wood != 40 {
			t.Errorf("turn %d: units %v, resources %+v", turn, counts, state.Resources)
		}
		queue := state.Buildings[0].ProductionQueue
		if turn < 3 {
			if len(queue) != 1 || queue[0].ItemID != 102 || queue[0].StaminaRemaining != 24-10*turn {
				t.Errorf("turn %d: queue %v, want piercing infantry with %d stamina remaining", turn, queue, 24-10*turn)
			}
		} else if len(queue) != 0 {
			t.Errorf("completed production retained queue %v", queue)
		}
	}
}

func TestProductionCancellationRefundsReservedFood(t *testing.T) {
	g := productionGame(t, 5, 1, "")
	p := g.Human(0)
	g.Submit(p, harness.OrderProduce([]uint64{centers(g, p)[0].InternalID}, 1))
	g.EndTurn()
	state := g.Self(p)
	if len(state.ActiveOrders) != 1 || state.Resources.Food != 250 {
		t.Fatalf("setup: orders %v, food %v", state.ActiveOrders, state.Resources.Food)
	}
	g.Submit(p, harness.Order(defs.OrderType_CancelOrder, nil, int64(state.ActiveOrders[0].InternalID), false))
	g.EndTurn()
	for turn := 0; turn < 2; turn++ {
		state = g.Self(p)
		if state.Resources.Food != 300 || len(state.Units) != 5 || len(centers(g, p)[0].ProductionQueue) != 0 {
			t.Errorf("cancelled production: food %v, units %d, queue %v", state.Resources.Food, len(state.Units), centers(g, p)[0].ProductionQueue)
		}
		g.EndTurn()
	}
}

func TestPartiallyProducedUnitCancellationRefundsOnce(t *testing.T) {
	g := economyGame(t, `{"food":100,"wood":100}`, `{"x":0,"y":0,"terrain":1,"zones":[{"x":0,"y":0,"building":{"id":22,"player":0},"units":[{"id":1,"player":0,"count":1}]}]},`+remoteVillager, `,"unlimited_population":true`)
	p := g.Human(0)
	g.Submit(p, harness.OrderProduce([]uint64{g.Self(p).Buildings[0].InternalID}, 102))
	g.EndTurn()
	state := g.Self(p)
	if len(state.ActiveOrders) != 1 || len(state.Buildings[0].ProductionQueue) != 1 || state.Buildings[0].ProductionQueue[0].StaminaRemaining != 4 || state.Resources.Food != 30 || state.Resources.Wood != 60 {
		t.Fatalf("production did not make partial progress: %+v", state)
	}
	g.Submit(p, harness.Order(defs.OrderType_CancelOrder, nil, int64(state.ActiveOrders[0].InternalID), false))
	for turn := 0; turn < 3; turn++ {
		g.EndTurn()
		state = g.Self(p)
		if state.Resources.Food != 100 || state.Resources.Wood != 100 || len(state.Units) != 1 || len(state.ActiveOrders) != 0 || len(state.Buildings[0].ProductionQueue) != 0 {
			t.Errorf("partial cancellation retained payment or spawned a unit: %+v", state)
		}
	}
}

func TestProductionQueueSpawnsOncePerOrder(t *testing.T) {
	g := productionGame(t, 1, 1, "")
	p := g.Human(0)
	b := centers(g, p)[0]
	order := harness.OrderProduce([]uint64{b.InternalID}, 1)
	g.Submit(p, order, order)
	for turn := 1; turn <= 3; turn++ {
		g.EndTurn()
		state := g.Self(p)
		wantUnits, wantQueue := 3, 0
		if turn == 1 {
			wantUnits, wantQueue = 2, 1
		}
		if len(state.Units) != wantUnits || len(centers(g, p)[0].ProductionQueue) != wantQueue || state.Resources.Food != 200 {
			t.Errorf("turn %d: units %d, queue %v, food %v; want %d units, %d queued, 200 food", turn, len(state.Units), centers(g, p)[0].ProductionQueue, state.Resources.Food, wantUnits, wantQueue)
		}
		for _, u := range state.Units {
			if u.UnitID != 1 || u.Space != b.Space || u.Zone != b.Zone {
				t.Errorf("unexpected produced unit %+v", u)
			}
		}
	}
}

func productionGame(t *testing.T, workers, centers int, extra string) *harness.Game {
	t.Helper()
	spaces := []string{fmt.Sprintf(`{"x":0,"y":0,"terrain":1,"zones":[{"x":0,"y":0,"building":{"id":1,"player":0},"units":[{"id":1,"player":0,"count":%d}]}]}`, workers)}
	for x := 1; x < centers; x++ {
		spaces = append(spaces, fmt.Sprintf(`{"x":%d,"y":0,"terrain":1,"zones":[{"x":0,"y":0,"building":{"id":1,"player":0}}]}`, x))
	}
	spaces = append(spaces, `{"x":-2,"y":0,"terrain":1,"zones":[{"x":0,"y":0,"units":[{"id":1,"player":1,"count":1}]}]}`)
	doc := `{"board_size":3,"players":2,"starting_bank":{"food":300,"wood":300,"stone":100,"gold":500}` + extra + `,"spaces":[` + strings.Join(spaces, ",") + `]}`
	fakeboard.UseConfig(t, testConfig, nil, map[string]string{"production": doc})
	return startGame(t, "custom:production", 1, 2)
}

func centers(g *harness.Game, human int) []harness.Building {
	buildings := g.Self(human).Buildings
	if len(buildings) == 2 && buildings[0].Space.X > buildings[1].Space.X {
		buildings[0], buildings[1] = buildings[1], buildings[0]
	}
	return buildings
}
