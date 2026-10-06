package invariants

import (
	"testing"

	"github.com/dgray001/gray_online/game/games/risq/tests/harness"
)

func checkState(t *testing.T, g *harness.Game, players int, capped bool) []object {
	t.Helper()
	views := snapshots(g, players)
	items := inventoryOf(t, views)
	for slot, view := range views {
		checkPlayer(t, self(view, slot), slot, capped)
		checkBoard(t, view, items)
		checkRegions(t, view)
	}
	return views
}
