package invariants

import (
	"math"
	"testing"
)

func checkPlayer(t *testing.T, player object, slot int, capped bool) {
	t.Helper()
	for key, value := range player["resources"].(object) {
		bound(t, key, value.(float64), 0, math.MaxFloat64)
	}
	checkAccounting(t, player)
	support := 0.0
	for _, building := range objects(player, "buildings") {
		if !building["under_construction"].(bool) {
			support += number(building, "population_support")
		}
	}
	limit := number(player, "population_limit")
	if limit != min(support, number(player, "max_population_limit")) {
		t.Errorf("player %d cap %v disagrees with support %v", slot, limit, support)
	}
	if capped && float64(len(objects(player, "units"))) > limit {
		t.Errorf("player %d exceeded population cap %v", slot, limit)
	}
	for _, kind := range []string{"units", "buildings"} {
		checkPlayerEntities(t, player, slot, kind)
	}
}
