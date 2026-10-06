package invariants

import "testing"

func checkGarrisons(t *testing.T, items inventory, locations map[float64]int) {
	t.Helper()
	for id, building := range items["buildings"] {
		garrison := building["garrisoned_units"].([]any)
		bound(t, "garrison size", float64(len(garrison)), 0, number(building, "garrison_capacity"))
		if building["has_garrisoned_units"] != (len(garrison) > 0) {
			t.Errorf("building %v garrison flag disagrees", id)
		}
		for _, entry := range garrison {
			unitID := entry.(float64)
			locations[unitID]++
			unit := items["units"][unitID]
			if unit == nil {
				t.Fatalf("garrison %v contains orphan unit %v", id, unitID)
			}
			if unit["garrisoned_in"] != id || unit["player_id"] != building["player_id"] {
				t.Errorf("unit %v has inconsistent garrison %v", unitID, id)
			}
			if unit["space_coordinate"] != nil || unit["zone_coordinate"] != nil {
				t.Errorf("unit %v has garrison and zone coordinates", unitID)
			}
		}
	}
}
