package shape

import (
	"testing"

	"github.com/dgray001/gray_online/game/games/risq/tests/harness"
)

func TestBoardPreservesAxialSlots(t *testing.T) {
	g := shapeGame(t, 3)
	rows := array(t, snapshot(g, g.Human(1), false)["spaces"])
	equal(t, len(rows), 9)
	count := 0
	for j, value := range rows {
		y := j - 4
		row := array(t, value)
		equal(t, len(row), 9-max(y, -y))
		for i, value := range row {
			if value == nil {
				continue
			}
			count++
			item := object(t, value)
			x := max(-4, -(4+y)) + i
			equal(t, item["coordinate"], map[string]any{"x": float64(x), "y": float64(y)})
			equal(t, item["coordinate_key"], float64(harness.SpaceKey(x, y)))
		}
	}
	equal(t, count, 9)
}
