package invariants

import "testing"

func TestNormalizationPreservesGameplayOrder(t *testing.T) {
	for _, key := range []string{"active_orders", "production_queue", "move_path", "target_priority", "spaces", "zones"} {
		t.Run(key, func(t *testing.T) {
			a := object{key: []any{object{"id": 1}, object{"id": 2}}}
			b := object{key: []any{object{"id": 2}, object{"id": 1}}}
			if canonical(normalize("", a)) == canonical(normalize("", b)) {
				t.Fatalf("normalization erased %s order", key)
			}
		})
	}
}

func TestNormalizationRetainsGameplayFields(t *testing.T) {
	for _, key := range []string{"internal_id", "health", "current_stamina", "ownership", "resources_left", "target_id", "player_id", "population_limit"} {
		a, b := object{key: 1}, object{key: 2}
		if canonical(normalize("", a)) == canonical(normalize("", b)) {
			t.Errorf("normalization discarded %s", key)
		}
	}
}
