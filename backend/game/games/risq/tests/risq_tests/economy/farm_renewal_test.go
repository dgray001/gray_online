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
