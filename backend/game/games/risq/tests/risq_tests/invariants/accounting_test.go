package invariants

import (
	"math"
	"testing"
)

func checkAccounting(t *testing.T, player object) {
	t.Helper()
	report, ok := player["turn_report"].(object)
	if !ok {
		return
	}
	bank := player["resources"].(object)
	seen := map[int]bool{}
	for _, flow := range objects(report, "resources") {
		category := int(number(flow, "category"))
		if category < 1 || category > 4 || seen[category] {
			t.Fatalf("invalid or duplicate resource category %d", category)
		}
		seen[category] = true
		key := []string{"food", "wood", "stone", "gold"}[category-1]
		want := number(flow, "start") + number(flow, "gathered") - number(flow, "spent")
		if math.Abs(number(bank, key)-want) > 0.000001 || math.Abs(number(flow, "final")-want) > 0.000001 {
			t.Errorf("%s bank %v disagrees with report %+v", key, bank[key], flow)
		}
	}
	if len(seen) != 4 {
		t.Errorf("resource report contains %d categories, want 4", len(seen))
	}
}
