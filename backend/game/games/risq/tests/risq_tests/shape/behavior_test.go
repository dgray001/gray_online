package shape

import (
	"testing"

	"github.com/gin-gonic/gin"
)

func TestChangedBehaviorWireValues(t *testing.T) {
	g := shapeGame(t, 4)
	owner, unit, center := g.Human(1), unitID(g, 1, 0, 11), centerID(g)
	g.Action(owner, "set-unit-behavior", gin.H{"internal_ids": []uint64{unit}, "stance": 1, "interrupt_current": true, "attack_back": false, "target_priority": []uint8{3, 1, 2}})
	g.Action(owner, "set-building-behavior", gin.H{"internal_ids": []uint64{center}, "auto_attack": false, "interrupt_current": true, "target_priority": []uint8{2, 3, 1}})
	for human := range 2 {
		p := player(t, snapshot(g, human, false), 1)
		u := checkShape(t, entity(t, p, "units", unit), unitShape+" "+privateUnitShape)
		equal(t, u["stance"], float64(1))
		equal(t, u["interrupt_current"], true)
		equal(t, u["attack_back"], false)
		equal(t, u["target_priority"], []any{float64(3), float64(1), float64(2)})
		b := checkShape(t, entity(t, p, "buildings", center), buildingShape+" "+privateBuildingShape)
		equal(t, b["auto_attack"], false)
		equal(t, b["interrupt_current"], true)
		equal(t, b["target_priority"], []any{float64(2), float64(3), float64(1)})
	}
}
