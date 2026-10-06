package shape

import (
	"testing"

	"github.com/dgray001/gray_online/game/games/risq/tests/harness"
)

func TestZoneKeysMatchFrontendCoordinates(t *testing.T) {
	g := shapeGame(t, 3)
	state := snapshot(g, g.Human(1), false)
	for _, x := range []int{0, 3} {
		target := space(t, state, x, 0)
		for j, value := range array(t, target["zones"]) {
			zy := j - 1
			for i, value := range array(t, value) {
				zx := max(-1, -(1+zy)) + i
				zone := object(t, value)
				equal(t, zone["coordinate"], map[string]any{"x": float64(zx), "y": float64(zy)})
				equal(t, zone["coordinate_key"], float64(harness.ZoneKey(x, 0, zx, zy)))
			}
		}
	}
}
