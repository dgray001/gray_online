package invariants

import "testing"

func checkSpace(t *testing.T, space object, items inventory, locations map[string]map[float64]int) {
	t.Helper()
	contents := map[string][]any{"units": {}, "buildings": {}, "resources": {}}
	for _, row := range space["zones"].([]any) {
		for _, value := range row.([]any) {
			if value == nil {
				continue
			}
			zone := value.(object)
			if zone["ownership"] != space["ownership"] {
				t.Error("zone ownership differs from space")
			}
			for _, unit := range objects(zone, "units") {
				checkZoneEntity(t, unit, space, zone, "units", items, locations)
				contents["units"] = append(contents["units"], unit)
			}
			if building, ok := zone["building"].(object); ok {
				checkZoneEntity(t, building, space, zone, "buildings", items, locations)
				contents["buildings"] = append(contents["buildings"], building)
			}
			if resource, ok := zone["resource"].(object); ok {
				id := number(resource, "internal_id")
				locations["resources"][id]++
				if locations["resources"][id] != 1 {
					t.Errorf("resource %v occurs in multiple zones", id)
				}
				checkResource(t, resource, space, zone)
				contents["resources"] = append(contents["resources"], resource)
			}
		}
	}
	for kind, entries := range contents {
		if canonical(normalize(kind, jsonValue(t, entries))) != canonical(normalize(kind, jsonValue(t, space[kind]))) {
			t.Errorf("space %v %s summary differs from zones", space["coordinate"], kind)
		}
	}
	checkOwnership(t, space)
}
