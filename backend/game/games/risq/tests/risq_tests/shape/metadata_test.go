package shape

import (
	"testing"

	"github.com/dgray001/gray_online/game/games/risq/internal/defs"
	"github.com/dgray001/gray_online/game/games/risq/tests/harness"
)

func TestBackgroundAndSpaceLinksWireShapes(t *testing.T) {
	saved15 := defs.UnitConfigs[15]
	t.Cleanup(func() { defs.UnitConfigs[15] = saved15 })
	defs.UnitConfigs[15] = defs.UnitConfigs[1]
	g := harness.NewGame(t, "custom:metadata", 1, 2)
	equal(t, len(array(t, snapshot(g, g.Human(0), true)["space_links"])), 0)
	state := checkShape(t, snapshot(g, g.Human(0), false), gameShape)
	equal(t, state["background_image"], "map.png")
	equal(t, state["background_top_left"], []any{float64(-4), float64(-2)})
	equal(t, state["background_top_right"], []any{float64(4), float64(2)})
	links := array(t, state["space_links"])
	equal(t, len(links), 1)
	link := checkShape(t, links[0], "from:o to:o direction:n to_direction:n")
	checkShape(t, link["from"], coordinateShape)
	checkShape(t, link["to"], coordinateShape)
	equal(t, link["from"], map[string]any{"x": float64(-4), "y": float64(0)})
	equal(t, link["to"], map[string]any{"x": float64(4), "y": float64(0)})
	equal(t, link["direction"], float64(3))
	equal(t, link["to_direction"], float64(0))
}
