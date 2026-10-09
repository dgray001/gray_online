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

func TestRegionsSixPartitionsRectangle(t *testing.T) {
	useScripts(t, map[string]string{"six": `[{"step":"shape","params":{"kind":"rectangle","size":{"rows":4,"cols":6}}},
		{"step":"regions_six","params":{"names":["Northwest","North"],"outer_bonus":20,"center_bonus":30}}]`})
	board := generateStartless(t, "six", 2)
	if len(board.Regions) != 6 || board.Regions[0].Name != "Northwest" || board.Regions[1].Name != "North" {
		t.Fatalf("unexpected regions: %v", board.Regions)
	}
	covered := map[uint]bool{}
	for i, region := range board.Regions {
		want := float64(20)
		if i%3 == 1 {
			want = 30
		}
		if len(region.Keys) != 4 || region.GoldBonus != want {
			t.Errorf("region %d: %d spaces, bonus %v", i, len(region.Keys), region.GoldBonus)
		}
		for key := range region.Keys {
			if covered[key] {
				t.Errorf("overlapping space %d", key)
			}
			covered[key] = true
		}
	}
	if len(covered) != 24 {
		t.Errorf("covered %d spaces, want 24", len(covered))
	}
	for _, space := range board.Inspect() {
		row := (space.Coord.Y + 2) / 2
		col := (space.Coord.X + int(math.Floor(float64(space.Coord.Y)/2)) + 3) / 2
		if !board.Regions[row*3+col].Keys[board.Space(space.Coord).Key()] {
			t.Errorf("space %v is in the wrong region", space.Coord)
		}
	}
}

func TestRegionsFourPartitionsTriangle(t *testing.T) {
	useScripts(t, map[string]string{"four": `[{"step":"shape","params":{"kind":"triangle","size":{"edge_length":6}}},
		{"step":"regions_four","params":{"names":["Center","Corner"],"outer_bonus":30,"center_bonus":60}}]`})
	board := generateStartless(t, "four", 2)
	if len(board.Regions) != 4 || board.Regions[0].Name != "Center" || board.Regions[1].Name != "Corner" {
		t.Fatalf("unexpected regions: %v", board.Regions)
	}
	covered := map[uint]bool{}
	for i, region := range board.Regions {
		wantSize, wantBonus := 6, float64(30)
		if i == 0 {
			wantSize, wantBonus = 10, 60
		}
		if len(region.Keys) != wantSize || region.GoldBonus != wantBonus {
			t.Errorf("region %d: %d spaces, bonus %v", i, len(region.Keys), region.GoldBonus)
		}
		for key := range region.Keys {
			if covered[key] {
				t.Errorf("overlapping space %d", key)
			}
			covered[key] = true
		}
	}
	if len(covered) != 28 {
		t.Errorf("covered %d spaces, want 28", len(covered))
	}
	membership := map[[2]int]int{{1, -2}: 0, {1, 1}: 0, {-2, 1}: 0, {0, 0}: 0, {-2, -2}: 1, {4, -2}: 2, {-2, 4}: 3}
	for _, space := range board.Inspect() {
		if region, ok := membership[[2]int{space.Coord.X, space.Coord.Y}]; ok {
			if !board.Regions[region].Keys[board.Space(space.Coord).Key()] {
				t.Errorf("space %v is not in region %d", space.Coord, region)
			}
		}
	}
}

func TestTriangleRegionBonusesFollowMapSize(t *testing.T) {
	script, err := os.ReadFile("../../config/maps/scripted/triangle.json")
	if err != nil {
		t.Fatal(err)
	}
	useScripts(t, map[string]string{"triangle": string(script)})
	for i, corner := range []float64{55, 90, 120, 155, 170, 210, 230, 300, 350} {
		board, err := fakeboard.GenerateWithSize("script:triangle", 2, defs.MapSize(i+1), 1)
		if err != nil {
			t.Fatal(err)
		}
		if len(board.Regions) != 4 {
			t.Fatalf("size %d: %d regions, want 4", i+1, len(board.Regions))
		}
		covered := map[uint]bool{}
		for j, region := range board.Regions {
			if len(region.Keys) == 0 {
				t.Errorf("size %d region %d is empty", i+1, j)
			}
			if j > 0 && len(region.Keys) != len(board.Regions[1].Keys) {
				t.Errorf("size %d: corner region sizes differ", i+1)
			}
			for key := range region.Keys {
				if covered[key] {
					t.Errorf("size %d: overlapping space %d", i+1, key)
				}
				covered[key] = true
			}
			want := corner
			if j == 0 {
				want *= 2
			}
			if region.GoldBonus != want || math.Mod(region.GoldBonus, 5) != 0 {
				t.Errorf("size %d region %d: bonus %v, want %v", i+1, j, region.GoldBonus, want)
			}
		}
		if len(covered) != len(board.Spaces()) {
			t.Errorf("size %d: regions cover %d of %d spaces", i+1, len(covered), len(board.Spaces()))
		}
	}
}

func TestRectangleRegionBonusesFollowMapSize(t *testing.T) {
	script, err := os.ReadFile("../../config/maps/scripted/rectangle.json")
	if err != nil {
		t.Fatal(err)
	}
	useScripts(t, map[string]string{"rectangle": string(script)})
	for i, outer := range []float64{35, 65, 85, 100, 115, 145, 165, 195, 230} {
		board, err := fakeboard.GenerateWithSize("script:rectangle", 2, defs.MapSize(i+1), 1)
		if err != nil {
			t.Fatal(err)
		}
		if len(board.Regions) != 6 {
			t.Fatalf("size %d: %d regions, want 6", i+1, len(board.Regions))
		}
		for j, region := range board.Regions {
			if math.Mod(region.GoldBonus, 5) != 0 {
				t.Errorf("size %d: bonus %v is not divisible by 5", i+1, region.GoldBonus)
			}
			if j%3 == 1 {
				if region.GoldBonus < 1.3*outer || region.GoldBonus > 1.5*outer {
					t.Errorf("size %d: middle-column bonus %v outside 30-50%% premium over %v", i+1, region.GoldBonus, outer)
				}
			} else if region.GoldBonus != outer {
				t.Errorf("size %d: outer bonus %v, want %v", i+1, region.GoldBonus, outer)
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
