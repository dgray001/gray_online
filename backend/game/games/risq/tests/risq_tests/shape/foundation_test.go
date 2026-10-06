package shape

import (
	"testing"

	"github.com/dgray001/gray_online/game/games/risq/tests/harness"
)

func TestFoundationWireShapeAndPrivacy(t *testing.T) {
	g := shapeGame(t, 4)
	unit, _ := queueWork(g)
	owner := g.Human(1)
	g.Submit(owner, harness.OrderBuild([]uint64{unit}, 2, 1, 0, 0, 0))
	g.EndTurn()
	p := player(t, snapshot(g, owner, false), 1)
	foundations := array(t, p["planned_foundations"])
	equal(t, len(foundations), 1)
	foundation := checkShape(t, foundations[0], "coordinate_key:n building_id:n display_name:s stamina_cost:n cost:o")
	checkShape(t, foundation["cost"], costShape)
	equal(t, foundation["coordinate_key"], float64(harness.ZoneKey(1, 0, 0, 0)))
	equal(t, foundation["building_id"], float64(2))
	equal(t, object(t, foundation["cost"])["wood"], float64(30))
	checkShape(t, player(t, snapshot(g, g.Human(0), false), 1), playerShape)
}
