package shape

import "testing"

func checkWorldCopies(t *testing.T, state, target map[string]any) {
	t.Helper()
	rows := array(t, target["zones"])
	equal(t, len(rows), 3)
	unitCount := 0
	for j, value := range rows {
		row := array(t, value)
		equal(t, len(row), []int{2, 3, 2}[j])
		for _, value := range row {
			zone := object(t, value)
			for _, kind := range []string{"building", "resource"} {
				if zone[kind] != nil {
					item := object(t, zone[kind])
					id := uint64(item["internal_id"].(float64))
					equal(t, entity(t, target, kind+"s", id), item)
					if kind == "building" {
						owner := player(t, state, int(item["player_id"].(float64)))
						equal(t, entity(t, owner, "buildings", id), item)
					} else {
						checkShape(t, item, resourceShape)
					}
				}
			}
			if zone["units"] != nil {
				units := array(t, zone["units"])
				unitCount += len(units)
				for _, value := range units {
					unit := object(t, value)
					id := uint64(unit["internal_id"].(float64))
					owner := player(t, state, int(unit["player_id"].(float64)))
					equal(t, entity(t, target, "units", id), unit)
					equal(t, entity(t, owner, "units", id), unit)
				}
			} else {
				unitCount += int(zone["unit_count"].(float64))
			}
		}
	}
	if target["units"] != nil {
		equal(t, len(array(t, target["units"])), unitCount)
	} else {
		equal(t, target["unit_count"], float64(unitCount))
	}
}
