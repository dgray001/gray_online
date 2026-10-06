package economy

import "testing"

func TestFarmDepletionKeepsTheBuilding(t *testing.T) {
	g := sourceGame(t, true)
	p := g.Human(0)
	before := g.Self(p).Resources.Food
	id := exhaustFarm(g, p)
	zone := farmZone(g, p)
	if zone.Building.InternalID != id || zone.Building.ResourcesLeft != 0 || zone.TerrainOverride != 25 {
		t.Errorf("depleted farm %+v, terrain %d; want same building, zero pool, terrain 25", zone.Building, zone.TerrainOverride)
	}
	if got := g.Self(p).Resources.Food - before; got != 200 {
		t.Errorf("farm yielded %v food, want 200", got)
	}
	g.EndTurn()
	if got := g.Self(p).Resources.Food - before; got != 200 {
		t.Errorf("farm yielded additional food after depletion: %v", got)
	}
	if buildings := g.Self(p).Buildings; len(buildings) != 1 || buildings[0].InternalID != id || buildings[0].ResourcesLeft != 0 {
		t.Errorf("owner buildings %+v, want the depleted farm", buildings)
	}
}
