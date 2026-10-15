package shape

import (
	"maps"
	"os"
	"path/filepath"
	"testing"

	"github.com/dgray001/gray_online/game/games/risq/internal/defs"
	"github.com/dgray001/gray_online/game/games/risq/tests/config"
	"github.com/dgray001/gray_online/game/games/risq/tests/harness"
)

var (
	testConfig = testconfig.Dir()
	pristine   *defs.ConfigSnapshot
)

const targetSpace = `{"x":0,"y":0,"terrain":1,"zones":[{"x":0,"y":0,"building":{"id":1,"player":1},"units":[{"id":1,"player":1,"count":1}]},{"x":1,"y":0,"terrain_override":24,"resource":41},{"x":-1,"y":0,"building":{"id":3,"player":1}},{"x":0,"y":-1,"units":[{"id":11,"player":1,"count":1}]}]}`
const scoutSpaces = `,{"x":-1,"y":0,"terrain":1,"zones":[{"x":0,"y":0,"units":[{"id":15,"player":0,"count":1}]}]},{"x":-2,"y":0,"terrain":1},{"x":-3,"y":0,"terrain":1},{"x":-4,"y":0,"terrain":1,"zones":[{"x":0,"y":0,"units":[{"id":15,"player":0,"count":1}]}]}`
const hiddenSpaces = `,{"x":1,"y":0,"terrain":1},{"x":2,"y":0,"terrain":1},{"x":3,"y":0,"terrain":1,"zones":[{"x":0,"y":0,"building":{"id":1,"player":1},"units":[{"id":1,"player":1,"count":1}]},{"x":1,"y":0,"resource":11}]},{"x":4,"y":0,"terrain":1}`
const regions = `,"regions":[{"name":"Border","gold_bonus":7,"spaces":[[0,0],[3,0]]},{"name":"Secret","gold_bonus":9,"spaces":[[4,0]]}]`

func TestMain(m *testing.M) {
	cleanup := setupConfigDir()
	defer cleanup()
	os.Exit(m.Run())
}

func setupConfigDir() func() {
	dir, _ := os.MkdirTemp("", "shape-*")
	mapDir := filepath.Join(dir, "maps", "custom")
	_ = os.MkdirAll(mapDir, 0o755)
	doc := `{"board_size":4,"players":2,"unlimited_population":true,"starting_techs":[4],"starting_bank":{"food":1000,"wood":1000,"stone":1000,"gold":1000},"spaces":[` + targetSpace + scoutSpaces + hiddenSpaces + `]` + regions + `}`
	_ = os.WriteFile(filepath.Join(mapDir, "shape.json"), []byte(doc), 0o644)
	metaDoc := `{"board_size":4,"players":2,"spaces":[` + targetSpace + scoutSpaces + hiddenSpaces + `],"background_image":{"name":"map.png","top_left":[-4,-2],"top_right":[4,2]},"connections":[{"from":[-4,0],"to":[4,0],"direction":3}]}`
	_ = os.WriteFile(filepath.Join(mapDir, "metadata.json"), []byte(metaDoc), 0o644)
	copyFiles(dir)
	_ = defs.LoadConfig(dir)
	return func() { _ = defs.LoadConfig(testConfig); _ = os.RemoveAll(dir) }
}

func copyFiles(dir string) {
	files := []string{"ai/default.json", "bonuses.json", "buildings.json", "region_names.json", "resources.json", "techs.json", "terrains.json", "units.json"}
	for _, f := range files {
		data, _ := os.ReadFile(filepath.Join(testConfig, f))
		dst := filepath.Join(dir, f)
		_ = os.MkdirAll(filepath.Dir(dst), 0o755)
		_ = os.WriteFile(dst, data, 0o644)
	}
}

func snapshotConfigs() func() {
	units := maps.Clone(defs.UnitConfigs)
	buildings := maps.Clone(defs.BuildingConfigs)
	techs := maps.Clone(defs.TechConfigs)
	terrains := maps.Clone(defs.TerrainConfigs)
	return func() {
		defs.UnitConfigs = units
		defs.BuildingConfigs = buildings
		defs.TechConfigs = techs
		defs.TerrainConfigs = terrains
	}
}

func shapeGame(t *testing.T, level uint8) *harness.Game {
	t.Helper()
	return configuredGame(t, level, func() {})
}

func configuredGame(t *testing.T, level uint8, configure func()) *harness.Game {
	t.Helper()
	t.Cleanup(snapshotConfigs())
	scout := defs.UnitConfigs[1]
	scout.Vision = defs.RisqVision{Space: max(level, defs.VisibilityGood), Edge_adjacent: level, Adjacent: level, Edge_opposite: level}
	defs.UnitConfigs[15] = scout
	villager := defs.UnitConfigs[1]
	villager.Turn_stamina = 1
	defs.UnitConfigs[1] = villager
	configure()
	return harness.NewGame(t, "custom:shape", 1, 2)
}

func player(t *testing.T, state map[string]any, id int) map[string]any {
	t.Helper()
	for _, value := range array(t, state["players"]) {
		item := object(t, value)
		if object(t, item["player"])["player_id"] == float64(id) {
			return item
		}
	}
	t.Fatalf("player %d missing", id)
	return nil
}

func space(t *testing.T, state map[string]any, x, y int) map[string]any {
	t.Helper()
	rows := array(t, state["spaces"])
	return object(t, array(t, rows[y+4])[x-max(-4, -(4+y))])
}

func entity(t *testing.T, owner map[string]any, collection string, id uint64) map[string]any {
	t.Helper()
	for _, value := range array(t, owner[collection]) {
		item := object(t, value)
		if item["internal_id"] == float64(id) {
			return item
		}
	}
	t.Fatalf("%s %d missing", collection, id)
	return nil
}

func unitID(g *harness.Game, slot, x int, kind uint32) uint64 {
	g.T.Helper()
	for _, unit := range g.Self(g.Human(slot)).Units {
		if unit.Space == (harness.Coord{X: x}) && unit.UnitID == kind {
			return unit.InternalID
		}
	}
	g.T.Fatalf("slot %d has no unit %d at (%d,0)", slot, kind, x)
	return 0
}

func centerID(g *harness.Game) uint64 {
	g.T.Helper()
	for _, building := range g.Self(g.Human(1)).Buildings {
		if building.BuildingID == 1 && building.Space == (harness.Coord{}) {
			return building.InternalID
		}
	}
	g.T.Fatal("target village center missing")
	return 0
}

func targetZoneContract(t *testing.T, zone map[string]any) string {
	t.Helper()
	coordinate := object(t, zone["coordinate"])
	x, y := coordinate["x"].(float64), coordinate["y"].(float64)
	contract := zoneShape
	if y == 0 && (x == 0 || x == -1) {
		contract += " building:o"
	}
	if y == 0 && x == 1 {
		contract += " resource:o terrain_override_display_name:s"
		equal(t, zone["terrain_override"], float64(24))
	} else {
		equal(t, zone["terrain_override"], float64(0))
	}
	return contract
}

func TestBoardPreservesAxialSlots(t *testing.T) {
	g := shapeGame(t, 3)
	rows := array(t, snapshot(g, g.Human(1), false)["spaces"])
	equal(t, len(rows), 9)
	count := 0
	for j, value := range rows {
		y := j - 4
		row := array(t, value)
		equal(t, len(row), 9-max(y, -y))
		for i, value := range row {
			if value == nil {
				continue
			}
			count++
			item := object(t, value)
			x := max(-4, -(4+y)) + i
			equal(t, item["coordinate"], map[string]any{"x": float64(x), "y": float64(y)})
			equal(t, item["coordinate_key"], float64(harness.SpaceKey(x, y)))
		}
	}
	equal(t, count, 9)
}

func TestZoneKeysMatchFrontendCoordinates(t *testing.T) {
	g := shapeGame(t, 3)
	state := snapshot(g, g.Human(1), false)
	for _, x := range []int{0, 3} {
		target := space(t, state, x, 0)
		for j, value := range array(t, target["zones"]) {
			zy := j - 1
			for i, value := range array(t, value) {
				zx := max(-1, -(1+zy)) + i
				zone := object(t, value)
				equal(t, zone["coordinate"], map[string]any{"x": float64(zx), "y": float64(zy)})
				equal(t, zone["coordinate_key"], float64(harness.ZoneKey(x, 0, zx, zy)))
			}
		}
	}
}

func TestRegionsOnlyIncludeExploredKeys(t *testing.T) {
	g := shapeGame(t, 3)
	state := snapshot(g, g.Human(0), false)
	regions := array(t, state["regions"])
	equal(t, len(regions), 1)
	region := checkShape(t, regions[0], "name:s gold_bonus:n spaces:a owner:n unexplored_spaces:n")
	equal(t, region["name"], "Border")
	equal(t, region["gold_bonus"], float64(7))
	equal(t, region["spaces"], []any{float64(harness.SpaceKey(0, 0))})
	equal(t, region["unexplored_spaces"], float64(1))
	ownerRegions := array(t, snapshot(g, g.Human(1), false)["regions"])
	equal(t, len(ownerRegions), 2)
	for _, value := range ownerRegions {
		equal(t, object(t, value)["unexplored_spaces"], float64(0))
	}
}

func TestTerrainStatsWireValues(t *testing.T) {
	for name, terrainType := range map[string]defs.TerrainType{"flatlands": defs.TerrainType_FLATLANDS, "hilly": defs.TerrainType_HILLY, "mountainous": defs.TerrainType_MOUNTAINOUS, "water": defs.TerrainType_WATER} {
		t.Run(name, func(t *testing.T) {
			g := configuredGame(t, 3, func() {
				config := defs.TerrainConfigs[1]
				config.Terrain_type = terrainType
				defs.TerrainConfigs[1] = config
			})
			stats := checkShape(t, space(t, snapshot(g, g.Human(0), false), 0, 0)["terrain_stats"], "intra_cost:n inter_cost:n impassable:b build_speed_modifier:n")
			cost := terrainType.MoveCost()
			equal(t, stats["intra_cost"], float64(cost.Intra_cost))
			equal(t, stats["inter_cost"], float64(cost.Inter_cost))
			equal(t, stats["impassable"], cost.Impassable)
			equal(t, stats["build_speed_modifier"], cost.Build_speed_modifier)
		})
	}
}

func TestGameAndPlayerWireShapes(t *testing.T) {
	g := shapeGame(t, 3)
	for human := range 2 {
		state := checkShape(t, snapshot(g, human, false), gameShape)
		base := checkShape(t, state["game_base"], "game_id:n game_type:n game_started:b game_ended:b persistant_history:b latest_update_id:n players:a viewers:a")
		for _, value := range array(t, base["players"]) {
			checkShape(t, value, identityShape)
		}
		for _, value := range array(t, state["players"]) {
			p := object(t, value)
			checkShape(t, p["player"], identityShape)
			contract := playerShape
			if object(t, p["player"])["player_id"] == float64(g.PlayerID(human)) {
				contract += " " + privatePlayerShape + " default_unit_stance:n default_unit_attack_back:b default_unit_interrupt_current:b default_unit_target_priority:a"
				checkShape(t, p["resources"], costShape)
				mercenaries := array(t, p["available_mercenaries"])
				equal(t, len(mercenaries), 2)
				for _, value := range mercenaries {
					checkProducible(t, value)
				}
			}
			checkShape(t, p, contract)
			equal(t, p["researched_techs"], map[string]any{"4": true})
		}
		for _, key := range []string{"background_top_left", "background_top_right"} {
			equal(t, state[key], []any{float64(0), float64(0)})
		}
	}
}

func TestViewerDoesNotInheritClientPrivateData(t *testing.T) {
	g := shapeGame(t, 4)
	queueWork(g)
	for human := range 2 {
		state := checkShape(t, snapshot(g, human, true), gameShape)
		equal(t, state["last_turn_replay"], nil)
		checkShape(t, state["game_base"], "game_id:n game_type:n game_started:b game_ended:b persistant_history:b latest_update_id:n players:a viewers:a player_actions:a viewer_updates:a")
		for _, value := range array(t, state["players"]) {
			p := checkShape(t, value, playerShape)
			for _, key := range []string{"units", "buildings", "active_orders"} {
				equal(t, len(array(t, p[key])), 0)
			}
		}
		for _, row := range array(t, state["spaces"]) {
			for _, value := range array(t, row) {
				if value != nil {
					s := checkShape(t, value, unexploredShape)
					equal(t, s["visibility"], float64(0))
				}
			}
		}
		equal(t, len(array(t, state["regions"])), 0)
	}
}


