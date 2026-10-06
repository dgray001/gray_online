package invariants

import "testing"

func checkZoneEntity(t *testing.T, entity, space, zone object, kind string, items inventory, locations map[string]map[float64]int) {
	t.Helper()
	id := number(entity, "internal_id")
	locations[kind][id]++
	owned := items[kind][id]
	if owned == nil {
		t.Fatalf("zone contains orphan %s %v", kind, id)
	}
	for _, copy := range []object{entity, owned} {
		if canonical(copy["space_coordinate"]) != canonical(space["coordinate"]) || canonical(copy["zone_coordinate"]) != canonical(zone["coordinate"]) {
			t.Errorf("%s %v coordinates disagree with containing zone", kind, id)
		}
		if copy["garrisoned_in"] != nil {
			t.Errorf("garrisoned unit %v also occurs in a zone", id)
		}
	}
	if entity["player_id"] != owned["player_id"] {
		t.Errorf("%s %v zone owner differs from inventory", kind, id)
	}
	if kind == "buildings" && canonical(normalize("garrisoned_units", jsonValue(t, entity["garrisoned_units"]))) != canonical(normalize("garrisoned_units", jsonValue(t, owned["garrisoned_units"]))) {
		t.Errorf("building %v garrison differs between zone and owner", id)
	}
}
