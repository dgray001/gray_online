package risq_mapgen_tests

import (
	"reflect"
	"testing"

	"github.com/dgray001/gray_online/game/game_utils"
	"github.com/dgray001/gray_online/game/games/risq/internal/defs"
	"github.com/dgray001/gray_online/game/games/risq/tests/harness"
)

func checkEstablishedPlayer(t *testing.T, player harness.PlayerState) {
	t.Helper()
	units, buildings := make(map[uint32]int), make(map[uint32]int)
	home := harness.Coord{}
	for _, unit := range player.Units {
		units[unit.UnitID]++
	}
	for _, building := range player.Buildings {
		buildings[building.BuildingID]++
		if building.BuildingID == 1 {
			home = building.Space
		}
	}
	if !reflect.DeepEqual(units, map[uint32]int{1: 6, 11: 2, 12: 1}) || !reflect.DeepEqual(buildings, map[uint32]int{1: 1, 2: 2, 3: 1, 21: 1}) {
		t.Errorf("Established units %v buildings %v", units, buildings)
	}
	for _, building := range player.Buildings {
		if building.UnderConstruction || building.CombatStats.Health != float64(building.CombatStats.MaxHealth) {
			t.Errorf("building %d did not start completed", building.BuildingID)
		}
		if building.BuildingID == 3 && building.ResourcesLeft != defs.BuildingConfigs[3].Gather.Starting_resources {
			t.Error("starting farm did not have its full resource pool")
		}
		want := uint(0)
		if building.BuildingID == 2 || building.BuildingID == 21 {
			want = 1
		}
		if distance := game_utils.AxialDistance(game_utils.Coordinate2D{X: home.X, Y: home.Y}, game_utils.Coordinate2D{X: building.Space.X, Y: building.Space.Y}); distance != want {
			t.Errorf("building %d distance %d, want %d", building.BuildingID, distance, want)
		}
	}
}

func TestEstablishedStartsOnEveryScriptMap(t *testing.T) {
	useTestScriptMaps(t)
	for _, name := range []string{"hexagon", "ring", "rectangle", "triangle"} {
		for _, balance := range []defs.StartBalance{defs.StartBalance_MIRRORED, defs.StartBalance_RANDOM} {
			g := harness.NewUnstartedGameWithSettings(t, "script:"+name, 1, 2, map[string]any{"starting_development": float64(defs.StartingDevelopment_ESTABLISHED), "start_balance": float64(balance)})
			g.Risq.StartGame()
			for player := 0; player < 2; player++ {
				checkEstablishedPlayer(t, g.Self(player))
			}
		}
	}
}
