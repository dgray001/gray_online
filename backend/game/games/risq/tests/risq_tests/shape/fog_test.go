package shape

import (
	"testing"

	"github.com/dgray001/gray_online/game/games/risq/internal/defs"
	"github.com/dgray001/gray_online/game/games/risq/tests/harness"
)

func TestFogWireShapeUsesFrozenContents(t *testing.T) {
	g := shapeGame(t, 4)
	queueWork(g)
	leaveTarget(g)
	reader, owner := g.Human(0), g.Human(1)
	remembered := space(t, snapshot(g, reader, false), 0, 0)
	equal(t, remembered["visibility"], float64(1))
	checkShape(t, remembered, exploredShape)
	checkFogContents(t, remembered)
	for _, value := range array(t, remembered["buildings"]) {
		building := checkShape(t, value, cachedBuildingShape)
		checkEntityDetails(t, building, "produces")
		for _, key := range []string{"turn_stamina", "current_stamina", "max_stamina"} {
			equal(t, building[key], float64(0))
		}
		equal(t, len(array(t, building["active_orders"])), 0)
		equal(t, len(array(t, building["production_queue"])), 0)
	}
	var deletes []defs.OrderFromFrontend
	for _, building := range g.Self(owner).Buildings {
		if building.Space == (harness.Coord{}) {
			deletes = append(deletes, harness.Order(defs.OrderType_BuildingDelete, []uint64{building.InternalID}, 0, false))
		}
	}
	deletes = append(deletes, harness.Order(defs.OrderType_UnitDelete, []uint64{unitID(g, 1, 0, 11)}, 0, false))
	g.Submit(owner, deletes...)
	g.EndTurn()
	g.EndTurn()
	live := space(t, snapshot(g, owner, false), 0, 0)
	equal(t, len(array(t, live["buildings"])), 0)
	equal(t, live["ownership"], float64(-1))
	frozen := space(t, snapshot(g, reader, false), 0, 0)
	checkShape(t, frozen, exploredShape)
	equal(t, frozen["ownership"], remembered["ownership"])
	equal(t, frozen["zones"], remembered["zones"])
	equal(t, frozen["resources"], remembered["resources"])
	for _, value := range array(t, remembered["buildings"]) {
		building := object(t, value)
		equal(t, entity(t, frozen, "buildings", uint64(building["internal_id"].(float64))), building)
	}
	previous := object(t, array(t, remembered["resources"])[0])["resources_left"].(float64)
	current := object(t, array(t, live["resources"])[0])["resources_left"].(float64)
	if current >= previous {
		t.Fatalf("live resource %v did not decrease from cached %v", current, previous)
	}
	enemy := player(t, snapshot(g, reader, false), 1)
	equal(t, len(array(t, enemy["units"])), 0)
	equal(t, len(array(t, enemy["buildings"])), 0)
	id := unitID(g, 0, -3, 15)
	g.Submit(reader, harness.OrderMove([]uint64{id}, -1, 0))
	for turn := 0; g.State(reader).Unit(id).Space != (harness.Coord{X: -1}); turn++ {
		if turn == 8 {
			t.Fatal("scout never restored vision")
		}
		g.EndTurn()
	}
	refreshed := space(t, snapshot(g, reader, false), 0, 0)
	equal(t, refreshed["visibility"], float64(4))
	equal(t, len(array(t, refreshed["buildings"])), 0)
	equal(t, refreshed["ownership"], float64(-1))
	equal(t, refreshed["resources"], space(t, snapshot(g, owner, false), 0, 0)["resources"])
}
