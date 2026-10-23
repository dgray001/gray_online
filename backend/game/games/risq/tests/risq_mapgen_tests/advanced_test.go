package risq_mapgen_tests

import (
	"reflect"
	"testing"

	"github.com/dgray001/gray_online/game/game_utils"
	"github.com/dgray001/gray_online/game/games/risq/internal/defs"
	"github.com/dgray001/gray_online/game/games/risq/tests/harness"
)

func checkAdvancedPlayer(t *testing.T, player harness.PlayerState) {
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
		t.Errorf("Advanced units %v cap %d, want 10/4/2/1 units and cap 25", units, player.PopulationLimit)
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
		t.Errorf("Advanced home buildings %v outer buildings %v", home_buildings, outer_buildings)
	}
}

func TestAdvancedStartsOnEveryScriptMap(t *testing.T) {
	t.Parallel()
	for i, name := range []string{"fixture_hexagon", "fixture_ring", "fixture_rectangle", "fixture_triangle"} {
		balance := defs.StartBalance(i%2 + 1)
		g := harness.NewUnstartedGameWithSettings(t, "script:"+name, 1, 2, map[string]any{"starting_development": float64(defs.StartingDevelopment_ADVANCED), "start_balance": float64(balance)})
		g.Risq.StartGame()
		for player := 0; player < 2; player++ {
			checkAdvancedPlayer(t, g.Self(player))
		}
	}
}
