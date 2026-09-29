package risq

import (
	"testing"

	"github.com/dgray001/gray_online/game/game_utils"
	"github.com/dgray001/gray_online/game/games/risq/internal/defs"
)

func slotTestUnit(id uint64) *RisqUnit {
	return &RisqUnit{orderableBase: orderableBase{internal_id: id}}
}

func TestGrantGatherSlotsKeepsHoldersThenLowestIds(t *testing.T) {
	source := &RisqResource{gather_capacity: 2}
	holder := slotTestUnit(9)
	holder.gather_slot = source
	newcomers := []*RisqUnit{slotTestUnit(5), slotTestUnit(3)}
	demands := []gatherDemand{{unit: newcomers[0]}, {unit: holder}, {unit: newcomers[1]}}
	granted, denied := grantGatherSlots(source, demands)
	if len(granted) != 2 || granted[0].unit != holder || granted[1].unit.internal_id != 3 {
		t.Fatalf("expected the holder then unit 3 to keep slots, got %v", granted)
	}
	if len(denied) != 1 || denied[0].unit.internal_id != 5 {
		t.Fatalf("expected unit 5 denied, got %v", denied)
	}
}

func TestGrantGatherSlotsUnderCapacityDeniesNobody(t *testing.T) {
	source := &RisqResource{gather_capacity: 4}
	granted, denied := grantGatherSlots(source, []gatherDemand{{unit: slotTestUnit(1)}, {unit: slotTestUnit(2)}})
	if len(granted) != 2 || len(denied) != 0 {
		t.Fatalf("expected both granted, got %d granted %d denied", len(granted), len(denied))
	}
}

// One space with a one-slot bush that an enemy villager already holds, and our villager ordered to it
func gatherContestGame(t *testing.T) (*GameRisq, *RisqUnit, *RisqUnit, *RisqResource, *RisqOrder) {
	t.Helper()
	if err := defs.LoadConfig("config"); err != nil {
		t.Fatalf("load config: %v", err)
	}
	r, mine, theirs := newTTKGame()
	space := r.spaces[0][0]
	center := &game_utils.Coordinate2D{}
	bush := createRisqResource(1, 1)
	bush.gather_capacity = 1
	space.setResource(center, bush)
	zone := space.getZone(center)
	holder := createRisqUnit(1, 1, theirs)
	newcomer := createRisqUnit(2, 1, mine)
	for _, u := range []*RisqUnit{holder, newcomer} {
		space.setUnit(center, u)
		r.units[u.internal_id] = u
		r.players[u.player_id].units[u.internal_id] = u
	}
	target := int64(zone.coordinate_key)
	holder.order_queue.receiveOrder(createRisqOrder(1, defs.OrderType_UnitGather, holder.player_id, map[uint64]Orderable{holder.internal_id: holder}, target, true))
	holder.gather_slot = bush
	order := createRisqOrder(2, defs.OrderType_UnitGather, newcomer.player_id, map[uint64]Orderable{newcomer.internal_id: newcomer}, target, true)
	newcomer.order_queue.receiveOrder(order)
	return r, holder, newcomer, bush, order
}

func TestGatherCapacityIsNotLeakedThroughOrders(t *testing.T) {
	r, _, newcomer, _, order := gatherContestGame(t)
	space := r.spaces[0][0]
	if !newcomer.orderReceivable(order, r) {
		t.Fatalf("a gather order to a known resource must be receivable even when its slots are taken")
	}
	// in fog the player can't see who holds the slots, so the order stands
	space.refreshCache(newcomer.player_id)
	space.visibility[newcomer.player_id] = defs.VisibilityFog
	if status := newcomer.orderStatus(order, r); status != OrderStatus_InProgress {
		t.Fatalf("in fog the order should stay in progress, got %v", status)
	}
	// once it can see the holder, it knows the bush is full
	space.visibility[newcomer.player_id] = defs.VisibilityGood
	if status := newcomer.orderStatus(order, r); status != OrderStatus_Cancelled {
		t.Fatalf("with good vision of a full bush the order should be cancelled, got %v", status)
	}
}

func TestGatherCapacityEnforcedAtExecute(t *testing.T) {
	r, holder, newcomer, bush, _ := gatherContestGame(t)
	for _, u := range []*RisqUnit{holder, newcomer} {
		u.intent.setGather(bush)
		u.intent.intent_cost = unitTickStaminaCost
	}
	r.gather_allotments = computeGatherAllotments([]Orderable{newcomer, holder})
	before := bush.resources_left
	newcomer.tickExecute(r)
	if bush.resources_left != before || newcomer.gather_slot != nil {
		t.Fatalf("the newcomer must get nothing from a full bush (left %v -> %v, slot %v)", before, bush.resources_left, newcomer.gather_slot)
	}
	holder.tickExecute(r)
	if bush.resources_left >= before || holder.gather_slot != bush {
		t.Fatalf("the holder should keep gathering and its slot")
	}
}
