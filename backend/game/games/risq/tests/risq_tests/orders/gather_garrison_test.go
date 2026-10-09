package orders

import (
	"testing"

	"github.com/dgray001/gray_online/game/games/risq/internal/defs"
	"github.com/dgray001/gray_online/game/games/risq/tests/harness"
	"github.com/gin-gonic/gin"
)

func TestBuildingGatherPointGarrisonsNewUnit(t *testing.T) {
	g := orderGame(t, richBank, "")
	p := g.Human(0)
	target := buildingID(g, p, 1)
	u := produceAtPoint(g, p, 1, 1, gin.H{"location_kind": 2, "location_id": harness.ZoneKey(0, 0, 0, 0), "object_type": 2, "object_id": target})
	if u.GarrisonedIn == nil || *u.GarrisonedIn != target {
		t.Fatalf("new unit was not garrisoned: %+v", u)
	}
	if u.CurrentStamina != u.TurnStamina+u.TurnStamina/2 || len(u.ActiveOrders) != 0 {
		t.Fatalf("direct garrison spent stamina or retained an order: %+v", u)
	}
}

func TestProductionGarrisonReservations(t *testing.T) {
	for _, reverse := range []bool{false, true} {
		for range 10 {
			useOrderConfig(t, richBank, "")
			center, villager := defs.BuildingConfigs[1], defs.UnitConfigs[1]
			center.Garrison_capacity, villager.Production_stamina = 1, 1
			defs.BuildingConfigs[1], defs.UnitConfigs[1] = center, villager
			g := harness.NewGame(t, "custom:orders", 1, 2)
			p := g.Human(0)
			building, units := buildingID(g, p, 1), unitIDs(g, p, 1)
			g.Action(p, "set-gather-point", gin.H{"building_id": building, "location_kind": 2, "location_id": harness.ZoneKey(0, 0, 0, 0), "object_type": 2, "object_id": building})
			orders := []defs.OrderFromFrontend{harness.OrderGarrison(units[:1], building), harness.OrderProduce([]uint64{building}, 1)}
			if reverse {
				orders[0], orders[1] = orders[1], orders[0]
			}
			g.Submit(p, orders...)
			g.EndTurn()
			after := unitIDs(g, p, 1)
			if len(after) != len(units)+1 {
				t.Fatal("newborn was not produced")
			}
			state := g.State(p)
			entrant, newborn := state.Unit(units[0]), state.Unit(after[len(after)-1])
			if entrant.GarrisonedIn == nil || *entrant.GarrisonedIn != building || newborn.GarrisonedIn != nil {
				t.Fatalf("newborn competed with existing entrant: entrant %+v, newborn %+v", entrant, newborn)
			}
			if len(entity(g, p, "buildings", building)["garrisoned_units"].([]uint64)) != 1 {
				t.Fatal("producer exceeded its garrison capacity")
			}
		}
	}
}
