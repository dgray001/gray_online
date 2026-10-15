package economy

import (
	"math"
	"testing"

	"github.com/dgray001/gray_online/game/games/risq/internal/defs"
	"github.com/dgray001/gray_online/game/games/risq/tests/harness"
)

func TestFarmRenewalPaysOnceAndRestoresGathering(t *testing.T) {
	g := sourceGame(t, true)
	p := g.Human(0)
	id := fastExhaustFarm(g, p)
	wood := g.Self(p).Resources.Wood
	u := g.Self(p).Units[0]
	g.Submit(p, harness.Order(defs.OrderType_UnitRenew, []uint64{u.InternalID}, int64(id), false))
	g.EndTurn()
	if got := g.Self(p).Resources.Wood; got != wood-60 {
		t.Errorf("renewal left %v wood, want %v", got, wood-60)
	}
	zone := farmZone(g, p)
	if zone.Building.InternalID != id || zone.Building.ResourcesLeft != 200 || zone.TerrainOverride != 24 {
		t.Errorf("renewed farm %+v, terrain %d; want same farm, pool 200, terrain 24", zone.Building, zone.TerrainOverride)
	}
	g.EndTurn()
	if got := g.Self(p).Resources.Wood; got != wood-60 {
		t.Errorf("renewal charged again: wood %v, want %v", got, wood-60)
	}
	food, pool := g.Self(p).Resources.Food, farmZone(g, p).Building.ResourcesLeft
	orders := g.Self(p).Units[0].ActiveOrders
	if len(orders) != 1 || orders[0].OrderType != uint8(defs.OrderType_UnitGather) {
		t.Fatalf("renewal did not switch to gathering: %+v", orders)
	}
	g.EndTurn()
	gain := g.Self(p).Resources.Food - food
	if left := farmZone(g, p).Building.ResourcesLeft; gain <= 0 || math.Abs(gain-(pool-left)) > 0.000001 {
		t.Errorf("renewed farm gained %v food with %v remaining", gain, left)
	}
}

func TestFarmRenewalPreservesQueuedMovement(t *testing.T) {
	g := sourceGame(t, true)
	p := g.Human(0)
	id := fastExhaustFarm(g, p)
	u := g.Self(p).Units[0]
	g.Submit(p,
		harness.Order(defs.OrderType_UnitRenew, []uint64{u.InternalID}, int64(id), false),
		harness.Order(defs.OrderType_UnitMoveZone, []uint64{u.InternalID}, harness.ZoneKey(0, 0, 1, 0), false))
	g.EndTurn()
	g.EndTurn()
	state := g.Self(p)
	if state.Units[0].Zone != (harness.Coord{X: 1}) || len(state.ActiveOrders) != 0 || farmZone(g, p).Building.ResourcesLeft != 200 {
		t.Fatalf("renewal displaced queued movement: %+v", state)
	}
}

func TestUnaffordableRenewalLeavesFarmDepleted(t *testing.T) {
	g := farmGame(t, 59, 1)
	p := g.Human(0)
	id := fastExhaustFarm(g, p)
	g.Submit(p, harness.Order(defs.OrderType_UnitRenew, []uint64{g.Self(p).Units[0].InternalID}, int64(id), false))
	g.EndTurn()
	if state := g.Self(p); state.Resources.Wood != 59 || farmZone(g, p).Building.ResourcesLeft != 0 || len(state.ActiveOrders) != 0 || len(state.Refusals()) != 1 || state.Refusals()[0] != "cannot afford renew" {
		t.Errorf("unaffordable renewal changed farm or balance: %+v", state)
	}
}

func TestFarmDepletionKeepsTheBuilding(t *testing.T) {
	g := sourceGame(t, true)
	p := g.Human(0)
	before := g.Self(p).Resources.Food
	id := exhaustFarm(g, p)
	zone := farmZone(g, p)
	if zone.Building.InternalID != id || zone.Building.ResourcesLeft != 0 || zone.TerrainOverride != 25 {
		t.Errorf("depleted farm %+v, terrain %d; want same building, zero pool, terrain 25", zone.Building, zone.TerrainOverride)
	}
	if got := g.Self(p).Resources.Food - before; got != 200 {
		t.Errorf("farm yielded %v food, want 200", got)
	}
	g.EndTurn()
	if got := g.Self(p).Resources.Food - before; got != 200 {
		t.Errorf("farm yielded additional food after depletion: %v", got)
	}
	if buildings := g.Self(p).Buildings; len(buildings) != 1 || buildings[0].InternalID != id || buildings[0].ResourcesLeft != 0 {
		t.Errorf("owner buildings %+v, want the depleted farm", buildings)
	}
}

func TestFarmGatherAndRenewRequireOwnership(t *testing.T) {
	g := economyGame(t, `{"wood":120}`, `{"x":0,"y":0,"terrain":1,"zones":[{"x":0,"y":0,"building":{"id":3,"player":0},"units":[{"id":1,"player":0,"count":1},{"id":1,"player":1,"count":1}]}]}`, "")
	p, enemy := g.Human(0), g.Human(1)
	u := g.Self(enemy).Units[0].InternalID
	if g.State(enemy).Space(0, 0).Visibility < int(defs.VisibilityPoor) {
		t.Fatal("enemy cannot see the farm")
	}
	g.Submit(enemy, harness.OrderGather([]uint64{u}, 0, 0, 0, 0))
	g.EndTurn()
	if refusals := g.Self(enemy).Refusals(); len(refusals) != 1 || refusals[0] != "not receivable" {
		t.Fatalf("non-owner gather refusals %v", refusals)
	}
	if state := g.Self(enemy); state.Resources.Food != 0 || farmZone(g, p).Building.ResourcesLeft != 200 {
		t.Fatalf("rejected gather credited food or drained the farm: resources %+v", state.Resources)
	}
	id := fastExhaustFarm(g, p)
	g.Submit(enemy, harness.Order(defs.OrderType_UnitRenew, []uint64{u}, int64(id), false))
	g.EndTurn()
	state := g.Self(enemy)
	if state.Resources.Wood != 120 || len(state.Refusals()) != 1 || state.Refusals()[0] != "not receivable" {
		t.Errorf("non-owner renewal: resources %+v, refusals %v", state.Resources, state.Refusals())
	}
	if farmZone(g, p).Building.ResourcesLeft != 0 || g.Self(p).Resources.Wood != 120 {
		t.Error("rejected renewal changed the owner's farm or wood")
	}
}

func TestPartialRenewalConsumesStaminaAndSharesPayment(t *testing.T) {
	for _, workers := range []int{1, 2} {
		g := farmGame(t, 120, workers)
		p := g.Human(0)
		id := fastExhaustFarm(g, p)
		original := defs.BuildingConfigs[3]
		config := original
		config.Gather.Renew_stamina = 48
		defs.BuildingConfigs[3] = config
		t.Cleanup(func() { defs.BuildingConfigs[3] = original })
		ids, stamina := []uint64{}, 0
		for _, u := range g.Self(p).Units {
			ids = append(ids, u.InternalID)
			stamina += u.CurrentStamina
		}
		g.Submit(p, harness.Order(defs.OrderType_UnitRenew, ids, int64(id), false))
		g.EndTurn()
		pool := farmZone(g, p).Building.ResourcesLeft
		if pool != 0 || g.Self(p).Resources.Wood != 60 || farmZone(g, p).Building.RenewStaminaRemaining != 48-stamina {
			t.Fatalf("%d renewers: pool %v, remaining %d, wood %v; want no refill and one payment", workers, pool, farmZone(g, p).Building.RenewStaminaRemaining, g.Self(p).Resources.Wood)
		}
		for _, u := range g.Self(p).Units {
			if u.CurrentStamina != u.TurnStamina {
				t.Errorf("renewal retained unspent stamina: %+v", u)
			}
		}
		for turn := 0; turn < 6 && farmZone(g, p).Building.ResourcesLeft < 200; turn++ {
			g.EndTurn()
		}
		if pool := farmZone(g, p).Building.ResourcesLeft; pool <= 0 || farmZone(g, p).Building.InternalID != id || g.Self(p).Resources.Wood != 60 || len(g.Self(p).ActiveOrders) != workers {
			t.Errorf("%d renewers failed to finish once: %+v", workers, g.Self(p))
		}
		for _, u := range g.Self(p).Units {
			if len(u.ActiveOrders) != 1 || u.ActiveOrders[0].OrderType != uint8(defs.OrderType_UnitGather) {
				t.Errorf("renewing villager did not gather: %+v", u.ActiveOrders)
			}
		}
	}
}
