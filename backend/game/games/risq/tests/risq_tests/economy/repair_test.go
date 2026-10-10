package economy

import (
	"math"
	"testing"

	"github.com/dgray001/gray_online/game/games/risq/internal/defs"
	"github.com/dgray001/gray_online/game/games/risq/tests/harness"
	"github.com/gin-gonic/gin"
)

func TestTickHistoryRepairSettlement(t *testing.T) {
	t.Run("unaffordable-repair", func(t *testing.T) {
		g, p := damagedRepairGame(t, `{"wood":0}`, 1)
		b, u := centerHousing(g, p), g.Self(p).Units[0]
		g.Submit(p, harness.Order(defs.OrderType_UnitRepair, []uint64{u.InternalID}, int64(b.InternalID), false))
		g.EndTurn()
		if centerHousing(g, p).CombatStats.Health != b.CombatStats.Health || len(g.Self(p).ActiveOrders) != 0 {
			t.Fatal("fixture repair was not cancelled without healing")
		}
		actions := g.State(p).Unit(u.InternalID).TickActions
		harness.AssertTickSpend(t, actions, 0)
		harness.RequireTickBlocked(t, actions, "cannot_afford_repair")
	})
	t.Run("damage-and-repair-settlement", func(t *testing.T) {
		g, p := damagedHousingGame(t)
		enemy := g.Human(1)
		b, u := centerHousing(g, p), g.Self(p).Units[0]
		ids := []uint64{}
		for _, attacker := range g.Self(enemy).Units {
			ids = append(ids, attacker.InternalID)
		}
		g.Submit(p, harness.Order(defs.OrderType_UnitRepair, []uint64{u.InternalID}, int64(b.InternalID), false))
		g.Submit(enemy, harness.Order(defs.OrderType_UnitAttackBuilding, ids, int64(b.InternalID), false))
		after := centerHousing(g, p)
		if after.CombatStats.Health <= 0 {
			t.Fatal("fixture building did not survive")
		}
		replay := harness.RequireReplay(t, g.State(p))
		health, frames := b.CombatStats.Health, 0
		for _, frame := range replay.Frames {
			if frame.Terminal {
				continue
			}
			damage, healing := 0.0, 0.0
			for _, effect := range frame.Effects {
				if effect.Kind == "damage" && effect.Target == (harness.TickTarget{Kind: "building", InternalID: b.InternalID}) {
					damage += effect.Amount
				}
			}
			for _, a := range after.TickActions {
				if a.Tick == frame.Tick {
					healing += a.Execute.Healing
				}
			}
			if damage == 0 && healing == 0 {
				continue
			}
			frames++
			found := false
			for _, state := range frame.Buildings {
				if state.InternalID != b.InternalID {
					continue
				}
				found = true
				if math.Abs(state.CombatStats.Health-(health+healing-damage)) > 0.0002 {
					t.Errorf("tick %d settled health %v, want %v", frame.Tick, state.CombatStats.Health, health+healing-damage)
				}
				health = state.CombatStats.Health
			}
			if !found {
				t.Errorf("tick %d has effects but no settled state", frame.Tick)
			}
		}
		if frames == 0 || math.Abs(health-after.CombatStats.Health) > 0.0002 {
			t.Errorf("recorded health %v, live health %v", health, after.CombatStats.Health)
		}
	})
	t.Run("newborn-gather-point-order", func(t *testing.T) {
		g := productionGame(t, 1, 2, "")
		p := g.Human(0)
		original, producer := g.Self(p).Units[0].InternalID, centers(g, p)[0].InternalID
		g.Action(p, "set-gather-point", gin.H{"building_id": producer, "location_kind": 1, "location_id": harness.SpaceKey(1, 0)})
		g.Submit(p, harness.OrderProduce([]uint64{producer}, 1))
		g.EndTurn()
		if len(g.Self(p).Units) != 2 {
			t.Fatal("fixture did not produce a unit")
		}
		for _, u := range g.Self(p).Units {
			if u.InternalID == original {
				continue
			}
			a := harness.RequireTickAction(t, u.TickActions, "move")
			if a.Tick < 3 || a.Order.Source != "gather_point" || a.Order.TargetID != harness.SpaceKey(1, 0) {
				t.Errorf("newborn synthetic order lost: %+v", a)
			}
			replay := harness.RequireReplay(t, g.State(p))
			if replay.UnitAt(u.InternalID, 0) != nil || replay.UnitAt(u.InternalID, replay.TickCount) == nil {
				t.Fatal("spawn boundary missing")
			}
		}
	})
	t.Run("research-and-spawn", func(t *testing.T) {
		g := productionGame(t, 1, 2, "")
		p := g.Human(0)
		original := g.Self(p).Units[0].InternalID
		config := defs.TechConfigs[1]
		config.Research_stamina = 10
		defs.TechConfigs[1] = config
		b := centers(g, p)
		g.Submit(p, harness.Order(defs.OrderType_BuildingResearch, []uint64{b[0].InternalID}, 1, false), harness.OrderProduce([]uint64{b[1].InternalID}, 1))
		g.EndTurn()
		if !g.Self(p).ResearchedTechs[1] || len(g.Self(p).Units) != 2 {
			t.Fatal("fixture did not complete research and production")
		}
		replay := harness.RequireReplay(t, g.State(p))
		for _, u := range g.Self(p).Units {
			settled := replay.UnitAt(u.InternalID, replay.TickCount)
			if settled == nil || settled.CombatStats.MaxHealth != 14 || settled.TurnStamina != 10 || settled.CombatStats.DefenseBlunt != 1 {
				t.Errorf("research settlement missing: %+v", settled)
			}
			if u.InternalID != original && replay.UnitAt(u.InternalID, 0) != nil {
				t.Fatal("spawn incorrectly present in baseline")
			}
		}
		for _, building := range centers(g, p) {
			harness.RequireTickAction(t, building.TickActions, "production")
			progress := 0
			for _, a := range building.TickActions {
				progress += a.Execute.Progress
			}
			if progress != 10 {
				t.Errorf("completion progress %d, want 10", progress)
			}
		}
	})
	t.Run("population-contention", func(t *testing.T) {
		g := productionGame(t, 9, 2, "")
		p := g.Human(0)
		b := centers(g, p)
		g.Submit(p, harness.OrderProduce([]uint64{b[0].InternalID}, 1), harness.OrderProduce([]uint64{b[1].InternalID}, 1))
		g.EndTurn()
		if len(g.Self(p).Units) != 10 {
			t.Fatal("fixture did not produce exactly one unit")
		}
		blocked := 0
		for _, building := range centers(g, p) {
			if len(building.ProductionQueue) > 0 {
				blocked++
				harness.RequireTickBlocked(t, building.TickActions, "population_cap")
			} else {
				a := harness.RequireTickAction(t, building.TickActions, "production")
				if a.Execute.StaminaSpent <= 0 {
					t.Errorf("winning producer has no recorded spend: %+v", a)
				}
			}
		}
		if blocked != 1 {
			t.Fatalf("fixture has %d blocked producers", blocked)
		}
	})
	t.Run("foundation-race", func(t *testing.T) {
		g := scenarioGame(t, `{"x":0,"y":0,"terrain":1,"zones":[{"x":0,"y":0,"units":[{"id":1,"player":0,"count":1},{"id":1,"player":1,"count":1}]}]},{"x":-1,"y":0,"terrain":1,"zones":[{"x":0,"y":0,"building":{"id":2,"player":0}}]},{"x":1,"y":0,"terrain":1,"zones":[{"x":0,"y":0,"building":{"id":2,"player":1}}]}`)
		for slot := range 2 {
			p := g.Human(slot)
			g.Submit(p, housingOrders(g, p, 0)...)
		}
		winners := 0
		for slot := range 2 {
			state := g.Self(g.Human(slot))
			if state.Resources.Wood == 120 {
				harness.RequireTickBlocked(t, state.Units[0].TickActions, "foundation_lost")
				harness.AssertTickSpend(t, state.Units[0].TickActions, 0)
			} else {
				winners++
				a := harness.RequireTickAction(t, state.Units[0].TickActions, "build")
				if a.Execute.Progress <= 0 || a.Execute.StaminaSpent <= 0 {
					t.Errorf("construction winner has no work: %+v", a)
				}
			}
		}
		if winners != 1 {
			t.Fatalf("fixture has %d foundation winners", winners)
		}
	})
	g, p := damagedHousingGame(t)
	b, u := centerHousing(g, p), g.Self(p).Units[0]
	wood := g.Self(p).Resources.Wood
	g.Submit(p, harness.Order(defs.OrderType_UnitRepair, []uint64{u.InternalID}, int64(b.InternalID), false))
	g.EndTurn()
	after := centerHousing(g, p)
	if after.CombatStats.Health <= b.CombatStats.Health {
		t.Fatal("fixture did not repair")
	}
	spent := 0
	for _, a := range g.Self(p).Units[0].TickActions {
		if a.Intent.Kind != "repair" {
			continue
		}
		spent += a.Execute.StaminaSpent
		if a.Tick < 1 || a.Execute.Healing != 0 || a.Execute.WoodSpent != 0 {
			t.Errorf("incorrect worker record: %+v", a)
		}
	}
	if spent != u.CurrentStamina {
		t.Errorf("repair history spent %d stamina, want %d", spent, u.CurrentStamina)
	}
	healing, cost := 0.0, 0.0
	for _, a := range after.TickActions {
		healing += a.Execute.Healing
		cost += a.Execute.WoodSpent
	}
	if math.Abs(healing-(after.CombatStats.Health-b.CombatStats.Health)) > 0.0002 || math.Abs(cost-(wood-g.Self(p).Resources.Wood)) > 0.00001 {
		t.Errorf("building settlement history: healing %v, wood %v", healing, cost)
	}
}

func TestRepairHealsProportionallyAndStopsAtFullHealth(t *testing.T) {
	g, p := damagedHousingGame(t)
	b := centerHousing(g, p)
	u := g.Self(p).Units[0]
	wood := g.Self(p).Resources.Wood
	wantHeal := 0.6 * float64(b.CombatStats.MaxHealth) / 14 * float64(u.CurrentStamina)
	g.Submit(p, harness.Order(defs.OrderType_UnitRepair, []uint64{u.InternalID}, int64(b.InternalID), false))
	g.EndTurn()
	after := centerHousing(g, p)
	healed := after.CombatStats.Health - b.CombatStats.Health
	if math.Abs(healed-wantHeal) > 0.0002 || after.CombatStats.Health >= float64(b.CombatStats.MaxHealth) {
		t.Errorf("first-turn healing %v, want %v and still damaged", healed, wantHeal)
	}
	wantCharge := (math.Round(after.CombatStats.Health*2500) - math.Round(b.CombatStats.Health*2500)) / 10000
	if got := wood - g.Self(p).Resources.Wood; math.Abs(got-wantCharge) > 0.00001 {
		t.Errorf("repair charged %v wood, want %v for the health restored", got, wantCharge)
	}
	for turn := 0; centerHousing(g, p).CombatStats.Health < float64(b.CombatStats.MaxHealth) && turn < 4; turn++ {
		g.EndTurn()
	}
	finished := centerHousing(g, p)
	if finished.InternalID != b.InternalID || finished.CombatStats.Health != float64(b.CombatStats.MaxHealth) {
		t.Errorf("repaired housing %+v, want same building at full health", finished)
	}
	wantTotal := (math.Round(float64(b.CombatStats.MaxHealth)*2500) - math.Round(b.CombatStats.Health*2500)) / 10000
	paid := wood - g.Self(p).Resources.Wood
	if math.Abs(paid-wantTotal) > 0.00001 {
		t.Errorf("total repair charge %v, want %v", paid, wantTotal)
	}
	g.EndTurn()
	if g.Self(p).Resources.Wood != wood-paid || centerHousing(g, p).CombatStats.Health != finished.CombatStats.Health {
		t.Error("completed repair kept charging or changed full health")
	}
}
