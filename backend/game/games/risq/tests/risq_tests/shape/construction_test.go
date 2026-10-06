package shape

import (
	"testing"

	"github.com/dgray001/gray_online/game/games/risq/internal/defs"
	"github.com/dgray001/gray_online/game/games/risq/tests/harness"
)

func TestUnderConstructionBuildingWireShape(t *testing.T) {
	g := configuredGame(t, 4, func() {
		unit := defs.UnitConfigs[1]
		unit.Turn_stamina = 10
		defs.UnitConfigs[1] = unit
		housing := defs.BuildingConfigs[2]
		housing.Build_stamina = 90
		defs.BuildingConfigs[2] = housing
	})
	g.Submit(g.Human(1), harness.OrderBuild([]uint64{unitID(g, 1, 0, 1)}, 2, 0, 0, 0, 1))
	g.EndTurn()
	for human := range 2 {
		owner := player(t, snapshot(g, human, false), 1)
		found := false
		for _, value := range array(t, owner["buildings"]) {
			building := object(t, value)
			if building["building_id"] == float64(2) {
				found = true
				checkShape(t, building, buildingShape+" "+privateBuildingShape)
				equal(t, building["under_construction"], true)
				equal(t, building["construction_stamina_total"], float64(90))
				if remaining := building["stamina_remaining"].(float64); remaining <= 0 || remaining >= 90 {
					t.Errorf("construction remaining %v, want partial progress", remaining)
				}
			}
		}
		if !found {
			t.Fatal("under-construction housing missing")
		}
	}
}
