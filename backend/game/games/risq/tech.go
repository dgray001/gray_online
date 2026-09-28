package risq

import "github.com/dgray001/gray_online/game/games/risq/internal/defs"

func applyTechBonus(u *RisqUnit, tech defs.TechConfig) {
	u.turn_stamina += tech.Bonus_turn_stamina
	u.cs.setMaxHealth(u.cs.max_health + tech.Bonus_max_health)
	if tech.HasTargetFilter() {
		return
	}
	u.cs.applyBonus(tech.Bonus)
}

func requiredTechMet(player *RisqPlayer, required_tech_id uint32) bool {
	return required_tech_id == 0 || player.researched_techs[required_tech_id]
}

type techCompletion struct {
	player_id int
	tech_id   uint32
}

func (r *GameRisq) completeResearch(player *RisqPlayer, tech_id uint32) {
	player.researched_techs[tech_id] = true
	player.report.recordTech(tech_id)
	tech, ok := defs.TechConfigs[tech_id]
	if !ok {
		return
	}
	for _, unit := range player.units {
		if tech.AppliesTo(unit.unit_id, unit.unitType()) {
			applyTechBonus(unit, tech)
		}
	}
	for _, unit_id := range tech.Unlocks_mercenary_ids {
		player.available_mercenaries[unit_id] = true
	}
}
