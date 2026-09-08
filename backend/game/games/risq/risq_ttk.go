package risq

import "sort"

type TTKResult struct {
	Turns    int
	Ticks    int
	Possible bool
}

// Simulates the real per-turn stamina regen and per-tick damage mechanics against a passive target
func computeTimeToKill(attacker_cs RisqCombatStats, attacker_turn_stamina int, target_max_health int, target_cs RisqCombatStats) TTKResult {
	if attacker_turn_stamina <= 0 {
		return TTKResult{}
	}
	health := float64(target_max_health)
	max_stamina := maxStaminaFor(attacker_turn_stamina)
	stamina := 0
	turns := 0
	ticks := 0
	for health > 0 && turns < 100000 {
		turns++
		stamina += attacker_turn_stamina
		if stamina > max_stamina {
			stamina = max_stamina
		}
		for stamina > 0 && health > 0 {
			cost := min(stamina, unitTickStaminaCost)
			damage := combatDamage(&attacker_cs, &target_cs, cost)
			if damage <= 0 {
				return TTKResult{}
			}
			health -= damage
			stamina -= cost
			ticks++
		}
	}
	if health > 0 {
		return TTKResult{}
	}
	return TTKResult{Turns: turns, Ticks: ticks, Possible: true}
}

func unitAttackerCombatStats(config UnitConfig) RisqCombatStats {
	cs := createRisqCombatStats()
	cs.attack_type = config.attack_type
	cs.attack_blunt = config.attack_blunt
	cs.attack_piercing = config.attack_piercing
	cs.penetration_blunt = config.penetration_blunt
	cs.penetration_piercing = config.penetration_piercing
	return cs
}

func ComputeUnitVsUnitTTK(attacker_unit_id uint32, target_unit_id uint32) TTKResult {
	attacker, ok := unitConfigs[attacker_unit_id]
	if !ok {
		return TTKResult{}
	}
	target, ok := unitConfigs[target_unit_id]
	if !ok {
		return TTKResult{}
	}
	target_cs := createRisqCombatStats()
	target_cs.defense_blunt = target.defense_blunt
	target_cs.defense_piercing = target.defense_piercing
	return computeTimeToKill(unitAttackerCombatStats(attacker), attacker.turn_stamina, target.max_health, target_cs)
}

func ComputeUnitVsBuildingTTK(attacker_unit_id uint32, target_building_id uint32) TTKResult {
	attacker, ok := unitConfigs[attacker_unit_id]
	if !ok {
		return TTKResult{}
	}
	target, ok := buildingConfigs[target_building_id]
	if !ok {
		return TTKResult{}
	}
	target_cs := createRisqCombatStats()
	target_cs.defense_blunt = target.defense_blunt
	target_cs.defense_piercing = target.defense_piercing
	return computeTimeToKill(unitAttackerCombatStats(attacker), attacker.turn_stamina, target.max_health, target_cs)
}

func AllUnitIDs() []uint32 {
	ids := make([]uint32, 0, len(unitConfigs))
	for id := range unitConfigs {
		ids = append(ids, id)
	}
	sort.Slice(ids, func(i, j int) bool { return ids[i] < ids[j] })
	return ids
}

func AllBuildingIDs() []uint32 {
	ids := make([]uint32, 0, len(buildingConfigs))
	for id := range buildingConfigs {
		ids = append(ids, id)
	}
	sort.Slice(ids, func(i, j int) bool { return ids[i] < ids[j] })
	return ids
}

func UnitName(id uint32) string {
	return unitConfigs[id].display_name
}

func BuildingName(id uint32) string {
	return buildingConfigs[id].display_name
}
