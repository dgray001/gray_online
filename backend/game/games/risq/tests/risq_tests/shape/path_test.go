package shape

import (
	"testing"

	"github.com/dgray001/gray_online/game/games/risq/tests/harness"
)

func TestMovePathWireShape(t *testing.T) {
	g := shapeGame(t, 3)
	id, human := unitID(g, 0, -1, 15), g.Human(0)
	g.Submit(human, harness.OrderMove([]uint64{id}, -4, 0))
	g.EndTurn()
	owner := player(t, snapshot(g, human, false), 0)
	u := entity(t, owner, "units", id)
	checkShape(t, u, unitShape+" "+privateUnitShape+" move_path:a")
	path := array(t, u["move_path"])
	if len(path) == 0 {
		t.Fatal("moving scout has no path")
	}
	for _, value := range path {
		step := checkShape(t, value, "space:o zone:o")
		checkShape(t, step["space"], coordinateShape)
		checkShape(t, step["zone"], coordinateShape)
	}
	last := object(t, path[len(path)-1])
	equal(t, object(t, last["space"])["x"], float64(-4))
}
