package invariants

import (
	"fmt"
	"testing"
)

func checkEntityBounds(t *testing.T, entity object) {
	t.Helper()
	label := fmt.Sprintf("entity %v", entity["internal_id"])
	stats := entity["combat_stats"].(object)
	bound(t, label+" health", number(stats, "health"), 0, number(stats, "max_health"))
	if number(stats, "health") == 0 {
		t.Errorf("%s survived with zero health", label)
	}
	bound(t, label+" stamina", number(entity, "current_stamina"), 0, number(entity, "max_stamina"))
	if _, ok := entity["building_id"]; !ok {
		return
	}
	bound(t, label+" construction", number(entity, "stamina_remaining"), 0, number(entity, "construction_stamina_total"))
	if _, ok := entity["resources_left"]; ok {
		bound(t, label+" resources", number(entity, "resources_left"), 0, number(entity, "resource_capacity"))
	}
}
