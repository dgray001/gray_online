package risq_mapgen_tests

import "testing"

func TestCustomMapBuildsRegions(t *testing.T) {
	regions := `,"regions":[{"name":"West","gold_bonus":3,"spaces":[[0,0],[-1,0]]},{"name":"East","spaces":[[1,0]]}]`
	board := mustGenerate(t, customDoc(1, regions, hex7), 1)
	if len(board.Regions) != 2 {
		t.Fatalf("%d regions, want 2", len(board.Regions))
	}
	if west := board.Regions[0]; west.Name != "West" || west.GoldBonus != 3 || len(west.Keys) != 2 {
		t.Errorf("west = %+v, want 2 spaces and a bonus of 3", west)
	}
	if east := board.Regions[1]; east.GoldBonus != 0 || len(east.Keys) != 1 {
		t.Errorf("east = %+v, want 1 space and no bonus", east)
	}
}

func TestCustomMapRejectsBadRegions(t *testing.T) {
	regions := func(body string) string { return `,"regions":[` + body + `]` }
	mustFail(t, customDoc(1, regions(`{"name":"A","spaces":[[9,9]]}`), hex7), 1, "is not on the map")
	mustFail(t, customDoc(1, regions(`{"name":"A","spaces":[[0,0]]},{"name":"B","spaces":[[0,0]]}`), hex7), 1, "already belongs to region")
	mustFail(t, customDoc(1, regions(`{"name":"A","spaces":[[0,0]]},{"name":"A","spaces":[[1,0]]}`), hex7), 1, "already exists")
}
