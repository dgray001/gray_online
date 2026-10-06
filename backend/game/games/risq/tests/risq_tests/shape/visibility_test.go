package shape

import (
	"fmt"
	"testing"
)

func TestVisibilityWireShapes(t *testing.T) {
	for _, level := range []uint8{0, 2, 3, 4} {
		t.Run(fmt.Sprint(level), func(t *testing.T) {
			g := shapeGame(t, level)
			state := snapshot(g, g.Human(0), false)
			target := space(t, state, 0, 0)
			if target["visibility"] != float64(level) {
				t.Fatalf("visibility %v, want %d", target["visibility"], level)
			}
			if level == 0 {
				checkShape(t, target, unexploredShape)
				return
			}
			contract, units := exploredShape+" unit_count:n", "unit_count:n"
			if level >= 3 {
				contract, units = exploredShape+" units:a", "units:a"
			} else {
				equal(t, target["unit_count"], float64(2))
			}
			checkShape(t, target, contract)
			checkShape(t, target["coordinate"], coordinateShape)
			for _, row := range array(t, target["zones"]) {
				for _, value := range array(t, row) {
					zone := object(t, value)
					contract := targetZoneContract(t, zone) + " " + units
					checkShape(t, zone, contract)
					checkShape(t, zone["coordinate"], coordinateShape)
				}
			}
		})
	}
}
