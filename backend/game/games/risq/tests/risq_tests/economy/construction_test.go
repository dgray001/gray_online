package economy

import "testing"

func TestConstructionSpendsOnceAndCarriesProgress(t *testing.T) {
	g := buildGame(t, 1)
	p := g.Human(0)
	wood := g.Self(p).Resources.Wood
	g.Submit(p, housingOrders(g, p, 0)...)
	g.EndTurn()
	buildings := g.Self(p).Buildings
	if len(buildings) != 1 {
		t.Fatalf("buildings %+v, want one housing", buildings)
	}
	b := buildings[0]
	if b.BuildingID != 2 || !b.UnderConstruction || b.StaminaRemaining != 6 {
		t.Errorf("first-turn housing %+v, want under construction with 6 stamina remaining", b)
	}
	if got := g.Self(p).Resources.Wood; got != wood-30 {
		t.Errorf("first-turn wood %v, want %v", got, wood-30)
	}
	g.EndTurn()
	buildings = g.Self(p).Buildings
	if len(buildings) != 1 || buildings[0].InternalID != b.InternalID || buildings[0].UnderConstruction || buildings[0].StaminaRemaining != 0 {
		t.Errorf("second-turn buildings %+v, want the same completed housing", buildings)
	}
	g.EndTurn()
	if got := g.Self(p).Resources.Wood; got != wood-30 {
		t.Errorf("wood after completion %v, want a single charge: %v", got, wood-30)
	}
}
