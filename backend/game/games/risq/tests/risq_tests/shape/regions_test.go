package shape

import (
	"testing"

	"github.com/dgray001/gray_online/game/games/risq/tests/harness"
)

func TestRegionsOnlyIncludeExploredKeys(t *testing.T) {
	g := shapeGame(t, 3)
	state := snapshot(g, g.Human(0), false)
	regions := array(t, state["regions"])
	equal(t, len(regions), 1)
	region := checkShape(t, regions[0], "name:s gold_bonus:n spaces:a owner:n")
	equal(t, region["name"], "Border")
	equal(t, region["gold_bonus"], float64(7))
	equal(t, region["spaces"], []any{float64(harness.SpaceKey(0, 0))})
	ownerRegions := array(t, snapshot(g, g.Human(1), false)["regions"])
	equal(t, len(ownerRegions), 2)
}
