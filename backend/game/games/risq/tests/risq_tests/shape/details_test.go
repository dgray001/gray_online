package shape

import "testing"

func checkEntityDetails(t *testing.T, entity map[string]any, actions string) {
	t.Helper()
	checkShape(t, entity["combat_stats"], combatShape)
	checkShape(t, entity["space_coordinate"], coordinateShape)
	checkShape(t, entity["zone_coordinate"], coordinateShape)
	for _, value := range array(t, entity[actions]) {
		checkProducible(t, value)
	}
}

func checkProducible(t *testing.T, value any) {
	t.Helper()
	item := object(t, value)
	contract := producibleShape
	switch item["kind"] {
	case float64(1):
		contract += " stats:o"
		checkShape(t, item["stats"], "health:n attack_type:n attack_blunt:n attack_piercing:n attack_range:n defense_blunt:n defense_piercing:n penetration_blunt:n penetration_piercing:n")
	case float64(2):
		contract += " affects_unit_ids:a affects_unit_types:a bonus:o unlocks_mercenaries:a"
		checkShape(t, item["bonus"], "max_health:n turn_stamina:n attack_blunt:n attack_piercing:n attack_magic:n defense_blunt:n defense_piercing:n defense_magic:n penetration_blunt:n penetration_piercing:n penetration_magic:n")
		for _, unlocked := range array(t, item["unlocks_mercenaries"]) {
			checkShape(t, unlocked, "id:n display_name:s")
		}
	case float64(3):
	default:
		t.Fatalf("unknown producible kind %v", item["kind"])
	}
	checkShape(t, item, contract)
	checkShape(t, item["cost"], costShape)
}
