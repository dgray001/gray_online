package risq

import "github.com/dgray001/gray_online/game/games/risq/internal/defs"

type buildingTechBonus struct {
	gather_capacity              int
	gather_speed                 int
	resource_capacity            float64
	renew_cost_multiplier        float64
	garrison_attack_cap_blunt    int
	garrison_attack_cap_piercing int
	garrison_attack_cap_magic    int
}

func (b *RisqBuilding) applyTechBonus(tech defs.TechConfig) {
	if !tech.AppliesToBuilding(b.building_id) {
		return
	}
	b.turn_stamina += tech.Bonus_turn_stamina
	b.attack_range = b.attack_range.WithBonus(tech.Bonus_range)
	b.splash_damage_percent += tech.Splash_damage_percent
	b.splash_max_targets += tech.Splash_max_targets
	b.cs.setMaxHealth(b.cs.max_health + tech.Bonus_max_health)
	b.garrison_capacity = uint16(max(0, int(b.garrison_capacity)+tech.Bonus_garrison_capacity))
	b.population_support = uint16(max(0, int(b.population_support)+tech.Bonus_population_support))
	b.tech_bonus.gather_capacity += tech.Bonus_gather_capacity
	b.tech_bonus.gather_speed += tech.Bonus_gather_speed
	b.tech_bonus.resource_capacity += tech.Bonus_resource_capacity
	b.tech_bonus.renew_cost_multiplier *= 1 - tech.Renew_cost_discount
	b.tech_bonus.garrison_attack_cap_blunt += tech.Bonus_garrison_attack_cap_blunt
	b.tech_bonus.garrison_attack_cap_piercing += tech.Bonus_garrison_attack_cap_piercing
	b.tech_bonus.garrison_attack_cap_magic += tech.Bonus_garrison_attack_cap_magic
	if !tech.HasTargetFilter() {
		b.cs.applyBonus(tech.Bonus)
	}
}

func (b *RisqBuilding) applyPlayerTechs(player *RisqPlayer) {
	for _, tech := range player.config.ActiveTechEffects(player.researchedTechIds()) {
		b.applyTechBonus(tech)
	}
	b.resources_left = b.resourceCapacity()
}

func (b *RisqBuilding) resourceCapacity() float64 {
	return b.config.Buildings[b.building_id].Gather.Starting_resources + b.tech_bonus.resource_capacity
}

func (b *RisqBuilding) renewCost() defs.RisqResourceCost {
	return b.config.Buildings[b.building_id].Gather.Renew_cost.Scale(b.tech_bonus.renew_cost_multiplier)
}
