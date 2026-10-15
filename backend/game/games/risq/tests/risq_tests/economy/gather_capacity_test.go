package economy

import (
	"fmt"
	"math"
	"reflect"
	"testing"

	"github.com/dgray001/gray_online/game"
	"github.com/dgray001/gray_online/game/games/risq/internal/defs"
	"github.com/dgray001/gray_online/game/games/risq/tests/harness"
	"github.com/dgray001/gray_online/game/games/risq/tests/harness/fakeboard"
	"github.com/gin-gonic/gin"
)

func TestTickHistoryGatherCapacity(t *testing.T) {
	for _, required := range []int{1, 48} {
		t.Run(fmt.Sprintf("renewal-%d", required), func(t *testing.T) {
			g := farmGame(t, 120, 2)
			p := g.Human(0)
			id := fastExhaustFarm(g, p)
			g.Lite = false
			g.Invalidate()
			config := defs.BuildingConfigs[3]
			config.Gather.Renew_stamina = required
			defs.BuildingConfigs[3] = config
			ids, available := []uint64{}, 0
			for _, u := range g.Self(p).Units {
				ids = append(ids, u.InternalID)
				available += u.CurrentStamina
			}
			g.Submit(p, harness.Order(defs.OrderType_UnitRenew, ids, int64(id), false))
			g.EndTurn()
			want := min(required, available)
			if farmZone(g, p).Building.RenewStaminaRemaining != required-want || g.Self(p).Resources.Wood != 60 {
				t.Fatal("fixture did not allocate renewal and pay once")
			}
			spent, progress, proposals := 0, 0, 0
			for _, u := range g.Self(p).Units {
				for _, a := range u.TickActions {
					if a.Intent.Kind != "renew" {
						continue
					}
					proposals++
					spent += a.Execute.StaminaSpent
					progress += a.Execute.Progress
				}
			}
			if proposals < 2 || spent != want || progress != want {
				t.Errorf("renewal recorded %d proposals, %d stamina, %d progress; want work %d", proposals, spent, progress, want)
			}
		})
	}
	t.Run("depleted-resource", func(t *testing.T) {
		g := gatherGame(t, 11, false, false)
		p := g.Human(0)
		submitGather(g, p)
		left, gained := 0.0, 0.0
		for turn := 0; turn < 60 && len(g.State(p).Space(0, 0).Resources) > 0; turn++ {
			left = g.State(p).Space(0, 0).Resources[0].ResourcesLeft
			before := g.Self(p).Resources.Wood
			g.EndTurn()
			gained = g.Self(p).Resources.Wood - before
		}
		if len(g.State(p).Space(0, 0).Resources) != 0 || math.Abs(gained-left) > 0.0001 {
			t.Fatal("fixture did not exhaust the resource")
		}
		g.Lite = false
		g.Invalidate()
		actions := g.Self(p).Units[0].TickActions
		harness.RequireTickAction(t, actions, "gather")
		recorded := 0.0
		for _, a := range actions {
			recorded += a.Execute.Gathered
		}
		if math.Abs(recorded-gained) > 0.0001 {
			t.Errorf("recorded gathering %v, want final pool %v", recorded, gained)
		}
		if len(g.Self(p).ActiveOrders) != 0 {
			t.Fatal("depleted gather order did not complete")
		}
	})
	g := sourceGame(t, false)
	g.Lite = false
	p := g.Human(0)
	units := g.Self(p).Units
	subjects := make([]uint64, len(units))
	for i, u := range units {
		subjects[i] = u.InternalID
	}
	before := g.Self(p).Resources.Wood
	g.Submit(p, harness.OrderGather(subjects, 0, 0, 0, 0))
	g.EndTurn()
	if g.Self(p).Resources.Wood <= before {
		t.Fatal("fixture did not gather")
	}
	blocked, executed := 0, 0
	for _, u := range g.Self(p).Units {
		if len(u.TickActions) == 0 {
			t.Errorf("unit %d missing tick history", u.InternalID)
		}
		for _, a := range u.TickActions {
			if a.Tick < 1 || a.Intent.Kind != "gather" {
				t.Errorf("incorrect gather intent: %+v", a)
			}
			if a.Execute.Outcome == "blocked" && a.Execute.Reason == "gather_capacity" && a.Execute.StaminaSpent == 0 {
				blocked++
			}
			if a.Execute.Outcome == "executed" && a.Execute.StaminaSpent > 0 {
				executed++
			}
		}
	}
	if blocked == 0 || executed == 0 {
		t.Errorf("capacity history: %d blocked, %d executed", blocked, executed)
	}
}

func TestTickHistorySurvivesReadsAndResets(t *testing.T) {
	t.Run("reconnect-and-turn-boundaries", func(t *testing.T) {
		g := sourceGame(t, false)
		g.Lite = false
		p := g.Human(0)
		g.Base.PlayerConnected(uint64(p + 1))
		u, turn := g.Self(p).Units[0], g.State(p).TurnNumber
		if g.State(p).LastTurnReplay != nil {
			t.Fatal("first turn has history before resolution")
		}
		g.Submit(p, harness.OrderGather([]uint64{u.InternalID}, 0, 0, 0, 0))
		g.EndTurn()
		replay := harness.RequireReplay(t, g.State(p))
		baseline, settled := replay.UnitAt(u.InternalID, 0), replay.UnitAt(u.InternalID, replay.TickCount)
		if replay.TurnNumber != turn || baseline == nil || baseline.CurrentStamina != u.CurrentStamina || settled == nil || settled.CurrentStamina != 0 {
			t.Fatal("baseline or tick stamina includes next-turn state")
		}
		if g.State(p).Unit(u.InternalID).CurrentStamina != u.TurnStamina {
			t.Fatal("fixture did not regenerate stamina")
		}
		client := uint64(p + 1)
		g.Base.PlayerConnected(client)
		game.Game_PlayerDisconnected(g.Risq, client)
		game.Game_PlayerReconnected(g.Risq, client)
		g.Base.ResendLastUpdate(client)
		select {
		case update := <-g.Base.Players[client].Updates:
			resent := harness.RequireReplay(t, harness.DecodeSnapshot(t, update.Content["game"]))
			if unit := resent.Unit(u.InternalID); unit == nil || !reflect.DeepEqual(unit.TickActions, g.State(p).Unit(u.InternalID).TickActions) {
				t.Fatal("resend or reconnect changed history")
			}
		default:
			t.Fatal("resend did not emit an update")
		}
	})
	g := sourceGame(t, false)
	g.Lite = false
	p := g.Human(0)
	u := g.Self(p).Units[0]
	g.Submit(p, harness.OrderGather([]uint64{u.InternalID}, 0, 0, 0, 0))
	g.EndTurn()
	first := g.State(p).Unit(u.InternalID).TickActions
	if len(first) == 0 {
		t.Fatal("resolved gather history missing")
	}
	if next := g.State(p).Unit(u.InternalID).TickActions; !reflect.DeepEqual(first, next) {
		t.Fatal("snapshot read changed history")
	}
	turn := g.State(p).TurnNumber
	g.Submit(p)
	if next := g.State(p); next.TurnNumber != turn || !reflect.DeepEqual(first, next.Unit(u.InternalID).TickActions) {
		t.Fatal("unresolved submit changed history")
	}
	g.Action(p, "unsubmit-orders", gin.H{})
	if next := g.State(p); next.TurnNumber != turn || !reflect.DeepEqual(first, next.Unit(u.InternalID).TickActions) {
		t.Fatal("unsubmit changed history")
	}
	orders := g.Self(p).ActiveOrders
	if len(orders) != 1 {
		t.Fatalf("fixture has %d active orders, want one", len(orders))
	}
	g.Submit(p, harness.Order(defs.OrderType_CancelOrder, nil, int64(orders[0].InternalID), false))
	g.EndTurn()
	for _, a := range g.State(p).Unit(u.InternalID).TickActions {
		if a.Intent.Kind == "gather" {
			t.Errorf("previous turn gather retained: %+v", a)
		}
	}
}

func TestGatherCapacityLimitsHarvest(t *testing.T) {
	g := sourceGame(t, false)
	p := g.Human(0)
	units := g.Self(p).Units
	subjects := make([]uint64, len(units))
	for i, u := range units {
		subjects[i] = u.InternalID
	}
	config := defs.ResourceConfigs[11]
	before := g.Self(p).Resources.Wood
	want := float64(units[0].CurrentStamina*config.Gather_capacity) * float64(config.Base_gather_speed) / 10
	g.Submit(p, harness.OrderGather(subjects, 0, 0, 0, 0))
	g.EndTurn()
	if got := g.Self(p).Resources.Wood - before; math.Abs(got-want) > 0.00001 {
		t.Errorf("three workers gained %v wood, want capacity-limited %v", got, want)
	}
	left := g.State(p).Space(0, 0).Resources[0].ResourcesLeft
	if math.Abs(left-(config.Starting_resources-want)) > 0.00001 {
		t.Errorf("cedar pool %v, want %v", left, config.Starting_resources-want)
	}
	idle := 0
	for _, u := range g.Self(p).Units {
		if u.CurrentStamina > units[0].CurrentStamina {
			idle++
		}
	}
	if idle != 1 {
		t.Errorf("%d workers retained unused stamina, want one", idle)
	}
}

func TestContestedDepletionSplitsFairly(t *testing.T) {
	var results [][2]float64
	for _, slots := range [][2]int{{0, 1}, {1, 0}} {
		g := gatherGame(t, 11, true, true)
		p0, p1 := g.Human(0), g.Human(1)
		category := defs.ResourceConfigs[11].Category
		before := [2]float64{balance(g, p0, category), balance(g, p1, category)}
		for _, slot := range slots {
			submitGather(g, g.Human(slot))
		}
		deplete(g, p0)
		gains := [2]float64{balance(g, p0, category) - before[0], balance(g, p1, category) - before[1]}
		if gains[0] != gains[1] {
			t.Errorf("submission order %v: unequal shares %v", slots, gains)
		}
		if total := gains[0] + gains[1]; total != defs.ResourceConfigs[11].Starting_resources {
			t.Errorf("submission order %v: total gathered %v, want %v", slots, total, defs.ResourceConfigs[11].Starting_resources)
		}
		results = append(results, gains)
	}
	if results[0] != results[1] {
		t.Errorf("submission order changed gains: %v", results)
	}
}

func TestGatherCreditsTheCorrectCategory(t *testing.T) {
	for name, id := range map[string]uint32{"food": 1, "wood": 11, "stone": 41} {
		t.Run(name, func(t *testing.T) {
			g := gatherGame(t, id, false, false)
			p := g.Human(0)
			config := defs.ResourceConfigs[id]
			before := *g.Self(p).Resources
			want := float64(g.Self(p).Units[0].CurrentStamina) * float64(config.Base_gather_speed) / 10
			submitGather(g, p)
			g.EndTurn()
			initial := []float64{before.Food, before.Wood, before.Stone}
			for i, category := range []defs.RisqResourceCategory{defs.RisqResourceCategory_FOOD, defs.RisqResourceCategory_WOOD, defs.RisqResourceCategory_STONE} {
				expected := 0.0
				if category == config.Category {
					expected = want
				}
				if got := balance(g, p, category) - initial[i]; math.Abs(got-expected) > 0.00001 {
					t.Errorf("category %d gained %v, want %v", category, got, expected)
				}
			}
			resources := g.State(p).Space(0, 0).Resources
			if len(resources) != 1 || math.Abs(resources[0].ResourcesLeft-(config.Starting_resources-want)) > 0.00001 {
				t.Errorf("remaining resources %+v, want %v", resources, config.Starting_resources-want)
			}
		})
	}
}

func TestDepletionConservesResources(t *testing.T) {
	g := gatherGame(t, 11, false, true)
	p := g.Human(0)
	config := defs.ResourceConfigs[11]
	before := balance(g, p, config.Category)
	submitGather(g, p)
	deplete(g, p)
	after := balance(g, p, config.Category)
	if got := after - before; got != config.Starting_resources {
		t.Errorf("gained %v wood, want the entire pool of %v", got, config.Starting_resources)
	}
	g.EndTurn()
	if got := balance(g, p, config.Category); got != after {
		t.Errorf("wood changed after depletion: %v to %v", after, got)
	}
}

func TestContestedGatherSatisfiesSmallerDemandFirst(t *testing.T) {
	for _, slots := range [][2]int{{0, 1}, {1, 0}} {
		doc := `{"board_size":1,"players":2,"starting_bank":{},"spaces":[{"x":0,"y":0,"terrain":1,"zones":[{"x":0,"y":0,"resource":11,"units":[{"id":1,"player":0,"count":1},{"id":2,"player":1,"count":1}]}]}]}`
		fakeboard.UseConfig(t, testConfig, nil, map[string]string{"unequal": doc})
		worker := defs.UnitConfigs[1]
		worker.Turn_stamina = 1
		defs.UnitConfigs[2] = worker
		resource := defs.ResourceConfigs[11]
		resource.Starting_resources, resource.Base_gather_speed = 4, 10
		defs.ResourceConfigs[11] = resource
		g := startGame(t, "custom:unequal", 1, 2)
		for _, slot := range slots {
			submitGather(g, g.Human(slot))
		}
		for slot, want := range []float64{3, 1} {
			state := g.Self(g.Human(slot))
			if state.Resources.Wood != want || len(state.Refusals()) != 0 {
				t.Errorf("submission order %v, slot %d: wood %v, refusals %v; want %v wood", slots, slot, state.Resources.Wood, state.Refusals(), want)
			}
		}
		space := g.State(g.Human(0)).Space(0, 0)
		if len(space.Resources) != 0 {
			t.Error("fully gathered resource remains on the board")
		}
	}
}
