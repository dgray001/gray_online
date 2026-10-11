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
		if building.BuildingID == 1 {
			home = building.Space
		}
	}
	if !reflect.DeepEqual(units, map[uint32]int{1: 10, 11: 4, 12: 2, 13: 1}) || player.PopulationLimit != 25 {
		t.Errorf("Advanced units %v cap %d, want 10/4/2/1 units and cap 25", units, player.PopulationLimit)
	}
	home_buildings, outer_buildings := make(map[uint32]int), make(map[uint32]int)
	for _, building := range player.Buildings {
		if building.UnderConstruction || building.CombatStats.Health != float64(building.CombatStats.MaxHealth) {
			t.Errorf("building %d did not start completed", building.BuildingID)
		}
		if building.BuildingID == 3 && building.ResourcesLeft != defs.BuildingConfigs[3].Gather.Starting_resources {
			t.Error("starting farm did not have its full resource pool")
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
	if !reflect.DeepEqual(home_buildings, map[uint32]int{1: 1, 3: 4, 2: 2}) || !reflect.DeepEqual(outer_buildings, map[uint32]int{2: 2, 21: 1, 11: 1, 22: 1}) {
		t.Errorf("Advanced home buildings %v outer buildings %v", home_buildings, outer_buildings)
	}
}

func TestAdvancedStartsOnEveryScriptMap(t *testing.T) {
	useTestScriptMaps(t)
	for _, name := range []string{"hexagon", "ring", "rectangle", "triangle"} {
		for _, balance := range []defs.StartBalance{defs.StartBalance_MIRRORED, defs.StartBalance_RANDOM} {
			g := harness.NewUnstartedGameWithSettings(t, "script:"+name, 1, 2, map[string]any{"starting_development": float64(defs.StartingDevelopment_ADVANCED), "start_balance": float64(balance)})
			g.Risq.StartGame()
			for player := 0; player < 2; player++ {
				checkAdvancedPlayer(t, g.Self(player))
			}
		}
	}
}
