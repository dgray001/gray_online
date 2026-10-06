package shape

import (
	"testing"

	"github.com/dgray001/gray_online/game/games/risq/internal/defs"
	"github.com/dgray001/gray_online/game/games/risq/tests/harness"
	"github.com/dgray001/gray_online/game/games/risq/tests/harness/fakeboard"
)

func TestBackgroundAndSpaceLinksWireShapes(t *testing.T) {
	doc := `{"board_size":4,"players":2,"spaces":[` + targetSpace + scoutSpaces + hiddenSpaces + `],"background_image":{"name":"map.png","top_left":[-4,-2],"top_right":[4,2]},"connections":[{"from":[-4,0],"to":[4,0],"direction":3}]}`
	fakeboard.UseConfig(t, shippedConfig, nil, map[string]string{"metadata": doc})
	defs.UnitConfigs[15] = defs.UnitConfigs[1]
	g := harness.NewGame(t, "custom:metadata", 1, 2)
	for _, viewer := range []bool{false, true} {
		state := checkShape(t, snapshot(g, g.Human(0), viewer), gameShape)
		equal(t, state["background_image"], "map.png")
		equal(t, state["background_top_left"], []any{float64(-4), float64(-2)})
		equal(t, state["background_top_right"], []any{float64(4), float64(2)})
		links := array(t, state["space_links"])
		equal(t, len(links), 1)
		link := checkShape(t, links[0], "from:o to:o direction:n")
		checkShape(t, link["from"], coordinateShape)
		checkShape(t, link["to"], coordinateShape)
		equal(t, link["from"], map[string]any{"x": float64(-4), "y": float64(0)})
		equal(t, link["to"], map[string]any{"x": float64(4), "y": float64(0)})
		equal(t, link["direction"], float64(3))
	}
}
