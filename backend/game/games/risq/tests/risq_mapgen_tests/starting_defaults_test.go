package risq_mapgen_tests

import (
	"github.com/dgray001/gray_online/game/games/risq/internal/defs"
	"reflect"
	"testing"
)

func TestAssetBlockReplacesEntirePreset(t *testing.T) {
	for _, c := range []struct {
		assets           string
		units, buildings map[uint32]int
	}{
		{`{"units":[{"unit_id":11,"count":2,"target":{"zone":"center"}}]}`, map[uint32]int{11: 2}, map[uint32]int{}},
		{`{"buildings":[{"building_id":201,"target":{"zone":"center"}}]}`, map[uint32]int{}, map[uint32]int{201: 1}},
	} {
		params := `{"pattern":"ring","area_size":0,"starting_distance":1,"terrain_id":1,"assets":` + c.assets + `,"resources":[]}`
		useScripts(t, map[string]string{"assets": `[` + hexagon2 + `,` + step("player_starts", params) + `]`})
		board := nomadBoard(t, "assets", defs.StartingDevelopment_ADVANCED, 1)
		for player := 0; player < 4; player++ {
			units, buildings, _ := board.Tally(player)
			if !reflect.DeepEqual(units, c.units) || !reflect.DeepEqual(buildings, c.buildings) {
				t.Errorf("assets %s player %d: units %v buildings %v", c.assets, player, units, buildings)
			}
		}
	}
}

func TestStartingResourceDefaultsAndOverrides(t *testing.T) {
	for _, c := range []struct {
		level     defs.MapResourceLevel
		resources string
		nodes     int
	}{
		{defs.MapResourceLevel_SPARSE, "", 14},
		{defs.MapResourceLevel_STANDARD, "", 17},
		{defs.MapResourceLevel_RICH, "", 23},
		{defs.MapResourceLevel_ABUNDANT, "", 25},
		{defs.MapResourceLevel_ABUNDANT, `,"resources":[]`, 0},
		{defs.MapResourceLevel_ABUNDANT, `,"resources":[{"resource_id":41,"count":2}]`, 2},
	} {
		params := `{"nomad":true,"assets":{}` + c.resources + `}`
		useScripts(t, map[string]string{"defaults": `[{"step":"shape","params":{"kind":"hexagon","size":{"radius":5}}},` + step("player_starts", params) + `]`})
		board, nodes := resourceLevelBoard(t, "defaults", c.level), 0
		for _, space := range board.Inspect() {
			for _, zone := range space.Zones {
				if zone.Resource != 0 {
					nodes++
				}
			}
		}
		if nodes != 4*c.nodes {
			t.Errorf("level %d resources %s: nodes %d, want %d", c.level, c.resources, nodes, 4*c.nodes)
		}
	}
}
