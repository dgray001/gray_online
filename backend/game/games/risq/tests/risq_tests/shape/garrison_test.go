package shape

import (
	"fmt"
	"testing"

	"github.com/dgray001/gray_online/game/games/risq/tests/harness"
	"github.com/gin-gonic/gin"
)

func garrisonSetup(g *harness.Game) (uint64, uint64) {
	g.T.Helper()
	unit, center, owner := unitID(g, 1, 0, 1), centerID(g), g.Human(1)
	g.Action(owner, "set-gather-point", gin.H{"building_id": center, "location_kind": 1, "location_id": harness.SpaceKey(1, 0), "object_type": 0, "object_id": 0})
	g.Submit(owner, harness.OrderGarrison([]uint64{unit}, center))
	g.EndTurn()
	if g.State(owner).Unit(unit).GarrisonedIn == nil {
		g.T.Fatal("villager did not garrison")
	}
	return unit, center
}

func TestGarrisonAndGatherPointVisibility(t *testing.T) {
	for _, level := range []uint8{2, 3, 4} {
		t.Run(fmt.Sprint(level), func(t *testing.T) {
			g := shapeGame(t, level)
			unit, center := garrisonSetup(g)
			owner := player(t, snapshot(g, g.Human(1), false), 1)
			u := checkShape(t, entity(t, owner, "units", unit), garrisonUnitShape+" "+privateUnitShape)
			equal(t, u["garrisoned_in"], float64(center))
			building := entity(t, owner, "buildings", center)
			checkShape(t, building, buildingShape+" "+privateBuildingShape+" gather_point:o")
			checkShape(t, building["gather_point"], "location_kind:n location_id:n object_type:n object_id:n")
			equal(t, building["garrisoned_units"], []any{float64(unit)})
			enemy := player(t, snapshot(g, g.Human(0), false), 1)
			b := entity(t, enemy, "buildings", center)
			contract := buildingShape
			if level == 4 {
				contract += " " + privateBuildingShape + " gather_point:o"
				equal(t, b["garrisoned_units"], []any{float64(unit)})
				equal(t, b["gather_point"], building["gather_point"])
			}
			checkShape(t, b, contract)
			equal(t, b["has_garrisoned_units"], true)
			for _, value := range array(t, enemy["units"]) {
				if object(t, value)["internal_id"] == float64(unit) {
					t.Error("enemy player list exposed a garrisoned unit")
				}
			}
		})
	}
}
