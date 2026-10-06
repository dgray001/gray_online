package invariants

import "testing"

type inventory map[string]map[float64]object

func inventoryOf(t *testing.T, views []object) inventory {
	t.Helper()
	items := inventory{"units": {}, "buildings": {}}
	for slot, view := range views {
		for kind, entries := range items {
			for _, entity := range objects(self(view, slot), kind) {
				id := number(entity, "internal_id")
				if entries[id] != nil {
					t.Fatalf("duplicate global %s id %v", kind, id)
				}
				entries[id] = entity
			}
		}
	}
	return items
}
