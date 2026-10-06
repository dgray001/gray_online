package risq_mapgen_tests

import "testing"

func TestRegionsSevenPartitionsTheBoard(t *testing.T) {
	useScripts(t, map[string]string{"regions": `[{"step":"shape","params":{"kind":"hexagon","size":6}},
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
