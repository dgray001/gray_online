package invariants

import (
	"github.com/dgray001/gray_online/game/games/risq/internal/defs"
	"github.com/dgray001/gray_online/game/games/risq/tests/harness"
)

func combatOrders(g *harness.Game, turn, slot int) []defs.OrderFromFrontend {
	if turn != 0 {
		return nil
	}
	return []defs.OrderFromFrontend{harness.OrderAttackUnit(unitIDs(g, slot, 13), unitIDs(g, 1-slot, 13)[0])}
}

func verifyCombat(g *harness.Game, turn int) {
	g.T.Helper()
	for slot := range 2 {
		units := unitIDs(g, slot, 13)
		if turn == 0 && (len(units) != 1 || g.State(g.Human(slot)).Unit(units[0]).CombatStats.Health >= 8) {
			g.T.Fatal("combat did not damage both fighters")
		}
		if turn == 3 && len(units) != 0 {
			g.T.Fatalf("mutual combat left fighters: %+v", g.Self(g.Human(slot)).Units)
		}
	}
}
