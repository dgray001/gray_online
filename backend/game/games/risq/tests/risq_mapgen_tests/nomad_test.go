package risq_mapgen_tests

import (
	"math/rand"
	"strings"
	"testing"

	"github.com/dgray001/gray_online/game/games/risq/internal/defs"
	"github.com/dgray001/gray_online/game/games/risq/internal/mapgen"
	"github.com/dgray001/gray_online/game/games/risq/tests/harness/fakeboard"
)

const nomadTestStarts = `{"pattern":"ring","area_size":0,"starting_distance":1,"terrain_id":1,"assets":{"units":[{"unit_id":1,"count":4,"target":{"zone":"center"}}],"buildings":[{"building_id":1,"target":{"zone":"center"}}]},"resources":[{"resource_id":1,"count":3,"zone":"edge"}]}`

func checkNomadUnitStacks(t *testing.T, board *fakeboard.Board, stack int) {
	t.Helper()
	units, resources := 0, 0
	for _, space := range board.Inspect() {
		for _, zone := range space.Zones {
			units += len(zone.Units)
			if zone.Resource != 0 {
				resources++
			}
			if len(zone.Units) != 0 && len(zone.Units) != stack {
				t.Errorf("unit stack %d, want %d", len(zone.Units), stack)
			}
		}
	}
	if units != 16 || resources != 12 {
		t.Errorf("units %d resources %d, want 16 and 12", units, resources)
	}
}

func TestNomadFlagOverridesDevelopment(t *testing.T) {
	for name, c := range map[string]struct {
		development defs.StartingDevelopment
		flag        string
		stack       int
	}{
		"auto_nomad":      {defs.StartingDevelopment_NOMAD, "", 1},
		"auto_standard":   {defs.StartingDevelopment_STANDARD, "", 4},
		"forced_nomad":    {defs.StartingDevelopment_STANDARD, `,"nomad":true`, 1},
		"forced_standard": {defs.StartingDevelopment_NOMAD, `,"nomad":false`, 4},
	} {
		t.Run(name, func(t *testing.T) {
			params := strings.TrimSuffix(nomadTestStarts, "}") + c.flag + "}"
			useScripts(t, map[string]string{"nomad": `[` + hexagon2 + `,` + step("player_starts", params) + `]`})
			board := nomadBoard(t, "nomad", c.development, 1)
			checkNomadUnitStacks(t, board, c.stack)
		})
	}
}

func TestNomadHasNoPreparedStartAreas(t *testing.T) {
	params := `{"nomad":true,"terrain_id":2,"area_size":100,"starting_distance":100,"starting_bank":{"food":5,"wood":6,"stone":7,"gold":8},"resources":[{"resource_id":1,"count":1}]}`
	useScripts(t, map[string]string{"nomad": `[` + hexagon2 + `,` + step("player_starts", params) + `,` + step("resource_place", `{"resource_id":1,"count":1,"outside_player_areas":true}`) + `]`})
	board := nomadBoard(t, "nomad", defs.StartingDevelopment_STANDARD, 1)
	resources := 0
	for _, space := range board.Inspect() {
		if space.Terrain != 1 {
			t.Error("nomad prepared start terrain")
		}
		for _, zone := range space.Zones {
			if zone.Resource != 0 {
				resources++
			}
		}
	}
	if resources != 5 || board.Banks[3] != (mapgen.StartingBank{Food: 5, Wood: 6, Stone: 7, Gold: 8}) {
		t.Errorf("resources %d banks %v, want 5 resources and map's bank override", resources, board.Banks)
	}
}

func TestEmptyAssetListsOverrideDevelopmentDefaults(t *testing.T) {
	params := `{"pattern":"ring","area_size":0,"starting_distance":1,"terrain_id":1,"assets":{},"resources":[]}`
	useScripts(t, map[string]string{"empty": `[` + hexagon2 + `,` + step("player_starts", params) + `]`})
	for _, development := range []defs.StartingDevelopment{defs.StartingDevelopment_NOMAD, defs.StartingDevelopment_STANDARD, defs.StartingDevelopment_ESTABLISHED, defs.StartingDevelopment_ADVANCED} {
		board := nomadBoard(t, "empty", development, 1)
		for player := 0; player < 4; player++ {
			units, buildings, _ := board.Tally(player)
			if len(units) != 0 || len(buildings) != 0 {
				t.Errorf("development %d player %d: defaults replaced empty lists", development, player)
			}
		}
	}
}

func nomadBoard(t *testing.T, name string, development defs.StartingDevelopment, seed int64) *fakeboard.Board {
	t.Helper()
	board := fakeboard.New()
	settings := mapgen.ResolveSettings(map[string]any{"starting_development": float64(development)}, 4)
	if err := mapgen.Generate(board, rand.New(rand.NewSource(seed)), 4, settings, "script:"+name); err != nil {
		t.Fatal(err)
	}
	return board
}

func TestNomadOnEveryScriptMap(t *testing.T) {
	t.Parallel()
	for _, name := range scriptPair(1) {
		board := nomadBoard(t, name, defs.StartingDevelopment_NOMAD, 1)
		for player := 0; player < 4; player++ {
			units, buildings, _ := board.Tally(player)
			if units[1] != 3 || len(units) != 1 || len(buildings) != 0 {
				t.Errorf("%s player %d: units %v buildings %v", name, player, units, buildings)
			}
		}
		if board.Dump() != nomadBoard(t, name, defs.StartingDevelopment_NOMAD, 1).Dump() {
			t.Errorf("%s nomad starts are not reproducible", name)
		}
		if board.Dump() == nomadBoard(t, name, defs.StartingDevelopment_NOMAD, 2).Dump() {
			t.Errorf("%s nomad starts did not vary with seed", name)
		}
	}
}
