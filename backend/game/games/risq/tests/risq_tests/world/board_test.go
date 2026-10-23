package world

import (
	"testing"

	"github.com/dgray001/gray_online/game/games/risq/tests/harness"
)

func TestRemovedSpacesDoNotExist(t *testing.T) {
	g := startGame(t, mapDoc("", spaceWith(0, 0, flatTerrain, unit(worker, 0))+","+spaceWith(1, 0, flatTerrain, unit(worker, 1))+","+spaceWith(3, 0, flatTerrain, "")))
	state := g.State(0)
	if state.SpaceCount() != 3 {
		t.Errorf("%d spaces in the payload, want only the 3 listed", state.SpaceCount())
	}
	if state.Space(2, 0) != nil || state.Space(0, 1) != nil {
		t.Error("an unlisted space appears in the payload")
	}
}

func TestLinksAndSeamsAreExportedToClients(t *testing.T) {
	spaces := spaceWith(-3, 0, flatTerrain, unit(worker, 0)) + "," + spaceWith(3, 0, flatTerrain, unit(worker, 1))
	connections := `,"connections":[{"from":[-3,0],"to":[3,0],"direction":3}]`
	state := startGame(t, mapDoc(connections, spaces)).State(0)
	if len(state.SpaceLinks) != 1 {
		t.Fatalf("%d links exported, want exactly 1 seam", len(state.SpaceLinks))
	}
	want := harness.SpaceLink{From: harness.Coord{X: -3}, To: harness.Coord{X: 3}, Direction: 3}
	if link := state.SpaceLinks[0]; link != want {
		t.Errorf("seam %+v, want %+v: direction 3 leaves From westward", link, want)
	}
}

func TestEconomyIsReportedOnlyWhenSomeoneCanHaveEconomicUnits(t *testing.T) {
	military := mapDoc("", spaceWith(0, 0, flatTerrain, unit(grunt, 0))+","+spaceWith(1, 0, flatTerrain, unit(grunt, 1)))
	if startGame(t, military).State(0).HasEconomy {
		t.Error("an grunt-only game reports an economy")
	}
	mixed := mapDoc("", spaceWith(0, 0, flatTerrain, unit(worker, 0))+","+spaceWith(1, 0, flatTerrain, unit(grunt, 1)))
	if !startGame(t, mixed).State(0).HasEconomy {
		t.Error("a game with a worker reports no economy")
	}
}

func TestProductionContainsEconomyAndMercenaries(t *testing.T) {
	spaces := spaceWith(0, 0, flatTerrain, unit(grunt, 0)) + "," + spaceWith(1, 0, flatTerrain, unit(grunt, 1))
	if startGame(t, mapDoc("", spaces)).State(0).HasProduction {
		t.Error("a military-only game reports production")
	}
	mixed := spaceWith(0, 0, flatTerrain, unit(worker, 0)) + "," + spaceWith(1, 0, flatTerrain, unit(grunt, 1))
	if !startGame(t, mapDoc("", mixed)).State(0).HasProduction {
		t.Error("an economy reports no production")
	}
	if !startGame(t, mapDoc(`,"starting_techs":[302]`, spaces)).State(0).HasProduction {
		t.Error("available mercenaries report no production")
	}
}

func TestBuildingsAreReportedOnlyWhenSomeoneCanHaveThem(t *testing.T) {
	military := mapDoc("", spaceWith(0, 0, flatTerrain, unit(grunt, 0))+","+spaceWith(1, 0, flatTerrain, unit(grunt, 1)))
	if startGame(t, military).State(0).HasBuildings {
		t.Error("an grunt-only game reports buildings")
	}
	mixed := mapDoc("", spaceWith(0, 0, flatTerrain, unit(worker, 0))+","+spaceWith(1, 0, flatTerrain, unit(grunt, 1)))
	if !startGame(t, mixed).State(0).HasBuildings {
		t.Error("a game with a worker, who can build, reports no buildings")
	}
}

func TestSpaceNameReplacesOnlyTheTerrainNameInTheSnapshot(t *testing.T) {
	named := `{"x":0,"y":0,"terrain":1,"name":"Brazil","zones":[{"x":0,"y":0,"units":[` + unit(worker, 0) + `]}]}`
	state := startGame(t, mapDoc("", named+","+spaceWith(1, 0, flatTerrain, unit(worker, 1)))).State(0)
	brazil, other := state.Space(0, 0), state.Space(1, 0)
	if brazil.DisplayName != "Brazil" {
		t.Errorf("named space shows %q, want Brazil", brazil.DisplayName)
	}
	if other.DisplayName == "" || other.DisplayName == "Brazil" || *brazil.TerrainID != *other.TerrainID {
		t.Errorf("unnamed space shows %q on terrain %v, want its terrain's name", other.DisplayName, other.TerrainID)
	}
}

func TestMercenaryRegionRuleIsExportedAndMapsCanTurnItOff(t *testing.T) {
	spaces := spaceWith(0, 0, flatTerrain, unit(worker, 0)) + "," + spaceWith(1, 0, flatTerrain, unit(worker, 1))
	if !startGame(t, mapDoc("", spaces)).State(0).MercenariesNeedRegion {
		t.Error("the default rule, which needs the whole region, was not exported")
	}
	if startGame(t, mapDoc(`,"mercenaries_need_region":false`, spaces)).State(0).MercenariesNeedRegion {
		t.Error("a map that turns the rule off still exports it as on")
	}
}

func TestOrdinaryBoardsExportNoLinks(t *testing.T) {
	state := startGame(t, mapDoc("", spaceWith(0, 0, flatTerrain, unit(worker, 0))+","+spaceWith(1, 0, flatTerrain, unit(worker, 1)))).State(0)
	if len(state.SpaceLinks) != 0 {
		t.Errorf("links = %v, want none", state.SpaceLinks)
	}
}
