package risq_mapgen_tests

import (
	"reflect"
	"testing"

	"github.com/dgray001/gray_online/game/game_utils"
	"github.com/dgray001/gray_online/game/games/risq/internal/defs"
	"github.com/dgray001/gray_online/game/games/risq/tests/harness"
)

const largeStartAssets = `{"units":[{"unit_id":11,"count":10,"target":{"zone":"center"}},{"unit_id":21,"count":4,"target":{"zone":"center"}},{"unit_id":31,"count":2,"target":{"zone":"center"}},{"unit_id":22,"count":1,"target":{"zone":"center"}}],"buildings":[{"building_id":201,"terrain_override":7,"target":{"zone":"center"}},{"building_id":203,"terrain_override":25,"target":{"zone":"edge"}},{"building_id":203,"terrain_override":25,"target":{"zone":"edge"}},{"building_id":203,"terrain_override":25,"target":{"zone":"edge"}},{"building_id":203,"terrain_override":25,"target":{"zone":"edge"}},{"building_id":202,"target":{"zone":"edge"}},{"building_id":202,"target":{"zone":"edge"}},{"building_id":202,"target":{"space_distance":1,"zone":"center"}},{"building_id":202,"target":{"space_distance":1,"zone":"center"}},{"building_id":207,"target":{"space_distance":1,"zone":"center"}},{"building_id":209,"target":{"space_distance":1,"zone":"center"}},{"building_id":205,"target":{"space_distance":1,"zone":"center"}}]}`

func checkLargeStartPlayer(t *testing.T, player harness.PlayerState) {
	t.Helper()
	units := make(map[uint32]int)
	home := harness.Coord{}
	for _, unit := range player.Units {
		units[unit.UnitID]++
	}
	for _, building := range player.Buildings {
		if building.BuildingID == 201 {
			home = building.Space
		}
	}
	if !reflect.DeepEqual(units, map[uint32]int{11: 10, 21: 4, 31: 2, 22: 1}) || player.PopulationLimit != 25 {
		t.Errorf("large start units %v cap %d, want 10/4/2/1 units and cap 25", units, player.PopulationLimit)
	}
	home_buildings, outer_buildings := make(map[uint32]int), make(map[uint32]int)
	for _, building := range player.Buildings {
		if building.UnderConstruction || building.CombatStats.Health != float64(building.CombatStats.MaxHealth) {
			t.Errorf("building %d did not start completed", building.BuildingID)
		}
		if building.BuildingID == 203 && building.ResourcesLeft != defs.BuildingConfigs[203].Gather.Starting_resources {
			t.Error("starting orchard did not have its full resource pool")
		}
		switch distance := game_utils.AxialDistance(game_utils.Coordinate2D{X: home.X, Y: home.Y}, game_utils.Coordinate2D{X: building.Space.X, Y: building.Space.Y}); distance {
		case 0:
			home_buildings[building.BuildingID]++
		case 1:
			outer_buildings[building.BuildingID]++
		default:
			t.Errorf("building %d distance %d, want home or outer ring", building.BuildingID, distance)
		}
	}
	if !reflect.DeepEqual(home_buildings, map[uint32]int{201: 1, 203: 4, 202: 2}) || !reflect.DeepEqual(outer_buildings, map[uint32]int{202: 2, 207: 1, 209: 1, 205: 1}) {
		t.Errorf("large start home buildings %v outer buildings %v", home_buildings, outer_buildings)
	}
}

func TestLargeStartAssetsOnEveryScriptMap(t *testing.T) {
	useFixtureScriptsWithAssets(t, largeStartAssets)
	for i, name := range sweptScripts {
		balance := defs.StartBalance(i%2 + 1)
		g := harness.NewUnstartedGameWithSettings(t, "script:"+name, 1, 2, map[string]any{"start_balance": float64(balance)})
		g.Risq.StartGame()
		for player := 0; player < 2; player++ {
			checkLargeStartPlayer(t, g.Self(player))
		}
	}
}
