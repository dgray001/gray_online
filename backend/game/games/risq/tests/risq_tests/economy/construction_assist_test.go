package economy

import "testing"

func TestConstructionAssistSharesBuildingAndCost(t *testing.T) {
	for _, workers := range []int{1, 2} {
		g := buildGame(t, workers)
		p := g.Human(0)
		wood := g.Self(p).Resources.Wood
		g.Submit(p, housingOrders(g, p, 0)...)
		g.EndTurn()
		buildings := g.Self(p).Buildings
		if len(buildings) != 1 {
			t.Fatalf("%d builders created %d buildings, want one", workers, len(buildings))
		}
		if got := g.Self(p).Resources.Wood; got != wood-30 {
			t.Errorf("%d builders left %v wood, want one charge: %v", workers, got, wood-30)
		}
		if buildings[0].BuildingID != 2 || buildings[0].UnderConstruction != (workers == 1) {
			t.Errorf("%d builders: housing %+v, want completed only with two builders", workers, buildings[0])
		}
		if len(g.Self(p).PlannedFoundations) != 0 {
			t.Error("materialized housing retained a planned foundation")
		}
		g.EndTurn()
		if after := g.Self(p); len(after.Buildings) != 1 || after.Buildings[0].InternalID != buildings[0].InternalID || after.Resources.Wood != wood-30 {
			t.Errorf("%d builders: next-turn state has duplicate housing or another charge", workers)
		}
	}
}
