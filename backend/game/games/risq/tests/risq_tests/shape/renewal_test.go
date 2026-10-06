package shape

import (
	"testing"

	"github.com/dgray001/gray_online/game/games/risq/internal/defs"
	"github.com/dgray001/gray_online/game/games/risq/tests/harness"
)

func TestRenewingFarmWireShape(t *testing.T) {
	g := configuredGame(t, 4, func() {
		farm := defs.BuildingConfigs[3]
		farm.Gather.Starting_resources = 0
		defs.BuildingConfigs[3] = farm
	})
	owner := player(t, snapshot(g, g.Human(1), false), 1)
	var farmID uint64
	for _, value := range array(t, owner["buildings"]) {
		farm := object(t, value)
		if farm["building_id"] == float64(3) {
			farmID = uint64(farm["internal_id"].(float64))
		}
	}
	if farmID == 0 {
		t.Fatal("farm missing")
	}
	g.Submit(g.Human(1), harness.Order(defs.OrderType_UnitRenew, []uint64{unitID(g, 1, 0, 1)}, int64(farmID), false))
	g.EndTurn()
	for human := range 2 {
		p := player(t, snapshot(g, human, false), 1)
		farm := checkShape(t, entity(t, p, "buildings", farmID), buildingShape+" "+privateBuildingShape+" "+farmShape)
		checkShape(t, farm["renew_cost"], costShape)
		equal(t, farm["resources_left"], float64(0))
		equal(t, farm["renewing"], true)
		equal(t, farm["display_name"], "Farm (expired)")
	}
}
