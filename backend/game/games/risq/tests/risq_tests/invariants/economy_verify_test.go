package invariants

import "github.com/dgray001/gray_online/game/games/risq/tests/harness"

func verifyEconomy(g *harness.Game, turn int) {
	g.T.Helper()
	for slot := range 3 {
		state := g.Self(g.Human(slot))
		if failures := state.Refusals(); len(failures) > 0 {
			g.T.Fatalf("economy turn %d refusals: %v", turn, failures)
		}
		if turn == 0 && g.State(g.Human(slot)).Unit(unitIDs(g, slot, 1)[2]).GarrisonedIn == nil {
			g.T.Fatal("garrison never exercised")
		}
		if turn == 1 && state.PopulationLimit != 10 {
			g.T.Fatal("housing did not complete")
		}
		if turn == 2 && g.State(g.Human(slot)).Unit(unitIDs(g, slot, 1)[2]).GarrisonedIn != nil {
			g.T.Fatal("ungarrison never exercised")
		}
		if turn == 6 && (len(state.Units) != 4 || len(state.Buildings) != 1 || !state.ResearchedTechs[1] || state.Resources.Wood <= 420) {
			g.T.Fatalf("economy incomplete: units %d, buildings %d, techs %v, wood %v", len(state.Units), len(state.Buildings), state.ResearchedTechs, state.Resources.Wood)
		}
	}
}
