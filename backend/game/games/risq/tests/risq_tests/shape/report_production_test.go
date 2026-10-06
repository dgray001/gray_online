package shape

import (
	"testing"

	"github.com/dgray001/gray_online/game/games/risq/internal/defs"
	"github.com/dgray001/gray_online/game/games/risq/tests/harness"
)

func TestReportBuildingAndResearchWireShapes(t *testing.T) {
	g := configuredGame(t, 3, func() {
		unit := defs.UnitConfigs[1]
		unit.Turn_stamina = 30
		defs.UnitConfigs[1] = unit
		housing := defs.BuildingConfigs[2]
		housing.Build_stamina = 1
		defs.BuildingConfigs[2] = housing
		tech := defs.TechConfigs[1]
		tech.Research_stamina = 1
		defs.TechConfigs[1] = tech
	})
	owner := g.Human(1)
	build := harness.OrderBuild([]uint64{unitID(g, 1, 0, 1)}, 2, 0, 0, 0, 1)
	research := harness.Order(defs.OrderType_BuildingResearch, []uint64{centerID(g)}, 1, false)
	g.Submit(owner, build, research)
	g.EndTurn()
	p := player(t, snapshot(g, owner, false), 1)
	production := object(t, object(t, p["turn_report"])["production"])
	buildings := array(t, production["buildings_built"])
	equal(t, len(buildings), 1)
	built := checkShape(t, buildings[0], "building_id:n space:o zone:o")
	checkShape(t, built["space"], coordinateShape)
	checkShape(t, built["zone"], coordinateShape)
	equal(t, built["building_id"], float64(2))
	equal(t, built["zone"], map[string]any{"x": float64(0), "y": float64(1)})
	equal(t, production["techs_researched"], []any{float64(1)})
	equal(t, p["researched_techs"], map[string]any{"1": true, "4": true})
}
