package risq_mapgen_tests

import (
	"reflect"
	"testing"

	"github.com/dgray001/gray_online/game/game_utils"
	"github.com/dgray001/gray_online/game/games/risq/internal/defs"
	"github.com/dgray001/gray_online/game/games/risq/tests/harness"
)

const smallStartAssets = `{"units":[{"unit_id":11,"count":6,"target":{"zone":"center"}},{"unit_id":21,"count":2,"target":{"zone":"center"}},{"unit_id":31,"count":1,"target":{"zone":"center"}}],"buildings":[{"building_id":201,"terrain_override":7,"target":{"zone":"center"}},{"building_id":203,"terrain_override":25,"target":{"zone":"edge"}},{"building_id":207,"target":{"space_distance":1,"zone":"center"}},{"building_id":202,"target":{"space_distance":1,"zone":"center"}},{"building_id":202,"target":{"space_distance":1,"zone":"center"}}]}`

func checkSmallStartPlayer(t *testing.T, player harness.PlayerState) {
	t.Helper()
	units, buildings := make(map[uint32]int), make(map[uint32]int)
	home := harness.Coord{}
	for _, unit := range player.Units {
		units[unit.UnitID]++
	}
	for _, building := range player.Buildings {
		buildings[building.BuildingID]++
		if building.BuildingID == 201 {
			home = building.Space
		}
	}
	if !reflect.DeepEqual(units, map[uint32]int{11: 6, 21: 2, 31: 1}) || !reflect.DeepEqual(buildings, map[uint32]int{201: 1, 202: 2, 203: 1, 207: 1}) {
		t.Errorf("small start units %v buildings %v", units, buildings)
	}
	for _, building := range player.Buildings {
		if building.UnderConstruction || building.CombatStats.Health != float64(building.CombatStats.MaxHealth) {
			t.Errorf("building %d did not start completed", building.BuildingID)
		}
		if building.BuildingID == 203 && building.ResourcesLeft != defs.BuildingConfigs[203].Gather.Starting_resources {
			t.Error("starting orchard did not have its full resource pool")
		}
		want := uint(0)
		if building.BuildingID == 202 || building.BuildingID == 207 {
			want = 1
		}
		if distance := game_utils.AxialDistance(game_utils.Coordinate2D{X: home.X, Y: home.Y}, game_utils.Coordinate2D{X: building.Space.X, Y: building.Space.Y}); distance != want {
			t.Errorf("building %d distance %d, want %d", building.BuildingID, distance, want)
		}
	}
}

func TestSmallStartAssetsOnEveryScriptMap(t *testing.T) {
	useFixtureScriptsWithAssets(t, smallStartAssets)
	for i, name := range sweptScripts {
		balance := defs.StartBalance(i%2 + 1)
		g := harness.NewUnstartedGameWithSettings(t, "script:"+name, 1, 2, map[string]any{"start_balance": float64(balance)})
		g.Risq.StartGame()
		for player := 0; player < 2; player++ {
			checkSmallStartPlayer(t, g.Self(player))
		}
	}
}
