package shape

import (
	"fmt"
	"testing"
)

func TestEnemyPrivateDataVisibility(t *testing.T) {
	for _, level := range []uint8{0, 2, 3, 4} {
		t.Run(fmt.Sprint(level), func(t *testing.T) {
			g := shapeGame(t, level)
			unit, center := queueWork(g)
			state := snapshot(g, g.Human(0), false)
			enemy := checkShape(t, player(t, state, 1), playerShape)
			equal(t, len(array(t, enemy["active_orders"])), 0)
			if level >= 2 {
				building := entity(t, enemy, "buildings", center)
				contract, count := buildingShape, 0
				if level == 4 {
					contract, count = contract+" "+privateBuildingShape, 1
				}
				checkShape(t, building, contract)
				equal(t, len(array(t, building["active_orders"])), count)
				equal(t, len(array(t, building["production_queue"])), count)
			} else {
				equal(t, len(array(t, enemy["buildings"])), 0)
			}
			if level >= 3 {
				u := entity(t, enemy, "units", unit)
				contract, count := unitShape, 0
				if level == 4 {
					contract, count = contract+" "+privateUnitShape, 1
					if u["move_path"] != nil {
						contract += " move_path:a"
					}
				}
				checkShape(t, u, contract)
				equal(t, len(array(t, u["active_orders"])), count)
			} else {
				equal(t, len(array(t, enemy["units"])), 0)
			}
			checkShape(t, space(t, state, 3, 0), unexploredShape)
		})
	}
}
