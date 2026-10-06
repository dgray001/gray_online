package invariants

import "testing"

func expectedOwner(space object) float64 {
	owners := map[float64]bool{}
	for _, building := range objects(space, "buildings") {
		owners[number(building, "player_id")] = true
	}
	if len(owners) == 0 {
		for _, unit := range objects(space, "units") {
			if number(unit, "unit_type") != 1 {
				owners[number(unit, "player_id")] = true
			}
		}
	}
	if len(owners) == 1 {
		for owner := range owners {
			return owner
		}
	}
	return -1
}

func checkOwnership(t *testing.T, space object) {
	t.Helper()
	if want := expectedOwner(space); space["ownership"] != want {
		t.Errorf("space %v owner %v, want %v", space["coordinate"], space["ownership"], want)
	}
}
