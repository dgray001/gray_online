package risq_mapgen_tests

import (
	"math"
	"os"
	"testing"

	"github.com/dgray001/gray_online/game/games/risq/internal/defs"
	"github.com/dgray001/gray_online/game/games/risq/tests/harness/fakeboard"
)

func TestRegionsSevenBonuses(t *testing.T) {
	cases := []struct {
		params        string
		center, outer float64
	}{
		{`{}`, 50, 30},
		{`{"center_bonus":0,"outer_bonus":7.5}`, 0, 7.5},
		{`{"center_bonus":"10 * map_size","outer_bonus":"board_size * 2"}`, 20, 4},
	}
	for _, c := range cases {
		useScripts(t, map[string]string{"bonuses": `[` + hexagon2 + `,{"step":"regions_seven","params":` + c.params + `}]`})
		board := generateStartless(t, "bonuses", 3)
		for i, region := range board.Regions {
			want := c.outer
			if i == 0 {
				want = c.center
			}
			if region.GoldBonus != want {
				t.Errorf("params %s: region %d bonus %v, want %v", c.params, i, region.GoldBonus, want)
			}
		}
	}
}

func TestRegionOuterBonusesFollowMapSize(t *testing.T) {
	scripts := map[string]string{}
	for shape, name := range map[string]string{"hexagon": "default", "ring": "ring"} {
		script, err := os.ReadFile("../../config/maps/scripted/" + name + ".json")
		if err != nil {
			t.Fatal(err)
		}
		scripts[shape] = string(script)
	}
	useScripts(t, scripts)
	bonuses := map[string][]float64{"hexagon": {30, 50, 70, 90, 90, 120, 120, 140, 180}, "ring": {30, 50, 50, 60, 90, 90, 105, 140, 160}}
	for shape, values := range bonuses {
		for i, want := range values {
			board, err := fakeboard.GenerateWithSize("script:"+shape, 2, defs.MapSize(i+1), 1)
			if err != nil {
				t.Fatalf("%s size %d: %v", shape, i+1, err)
			}
			for j, region := range board.Regions {
				if math.Mod(region.GoldBonus, 5) != 0 {
					t.Errorf("%s size %d: bonus %v is not divisible by 5", shape, i+1, region.GoldBonus)
				}
				if shape == "hexagon" && j == 0 {
					if region.GoldBonus < 1.5*want || region.GoldBonus > 1.8*want {
						t.Errorf("size %d: center bonus %v outside 50-80%% premium over %v", i+1, region.GoldBonus, want)
					}
					continue
				}
				if region.GoldBonus != want {
					t.Errorf("%s size %d: outer bonus %v, want %v", shape, i+1, region.GoldBonus, want)
				}
			}
		}
	}
}

func TestRegionsSevenPartitionsTheBoard(t *testing.T) {
	useScripts(t, map[string]string{"regions": `[{"step":"shape","params":{"kind":"hexagon","size":{"radius":6}}},
		{"step":"regions_seven","params":{"names":["Core","North"]}}]`})
	board := generateStartless(t, "regions", 2)
	if len(board.Regions) != 7 || board.Regions[0].Name != "Core" || board.Regions[1].Name != "North" {
		t.Fatalf("regions = %v, want 7 starting with Core and North", board.Regions)
	}
	covered, names := map[uint]bool{}, map[string]bool{}
	for _, region := range board.Regions {
		names[region.Name] = true
		if len(region.Keys) == 0 {
			t.Errorf("region %s is empty", region.Name)
		}
		for key := range region.Keys {
			covered[key] = true
		}
	}
	if len(names) != 7 {
		t.Errorf("%d distinct region names, want 7", len(names))
	}
	if len(covered) != len(board.Inspect()) {
		t.Errorf("regions cover %d of %d spaces", len(covered), len(board.Inspect()))
	}
}
