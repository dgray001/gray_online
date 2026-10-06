package invariants

import "testing"

func checkPlayerEntities(t *testing.T, player object, slot int, kind string) {
	t.Helper()
	seen := map[float64]bool{}
	for _, entity := range objects(player, kind) {
		id := number(entity, "internal_id")
		if seen[id] || number(entity, "player_id") != float64(slot) {
			t.Errorf("%s %v duplicated or assigned to wrong player %d", kind, id, slot)
		}
		seen[id] = true
		checkEntityBounds(t, entity)
	}
}
