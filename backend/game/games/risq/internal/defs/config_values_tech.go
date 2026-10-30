package defs

func bonusFields(prefix string) fieldTable[TechConfig] {
	return fieldTable[TechConfig]{
		prefix + "attack_blunt":         func(t TechConfig) float64 { return float64(t.Bonus.Attack_blunt) },
		prefix + "attack_piercing":      func(t TechConfig) float64 { return float64(t.Bonus.Attack_piercing) },
		prefix + "attack_magic":         func(t TechConfig) float64 { return float64(t.Bonus.Attack_magic) },
		prefix + "defense_blunt":        func(t TechConfig) float64 { return float64(t.Bonus.Defense_blunt) },
		prefix + "defense_piercing":     func(t TechConfig) float64 { return float64(t.Bonus.Defense_piercing) },
		prefix + "defense_magic":        func(t TechConfig) float64 { return float64(t.Bonus.Defense_magic) },
		prefix + "penetration_blunt":    func(t TechConfig) float64 { return float64(t.Bonus.Penetration_blunt) },
		prefix + "penetration_piercing": func(t TechConfig) float64 { return float64(t.Bonus.Penetration_piercing) },
		prefix + "penetration_magic":    func(t TechConfig) float64 { return float64(t.Bonus.Penetration_magic) },
	}
}

var techFields = merge(
	fieldTable[TechConfig]{
		"research_stamina":                     func(t TechConfig) float64 { return float64(t.Research_stamina) },
		"bonus_max_health":                     func(t TechConfig) float64 { return float64(t.Bonus_max_health) },
		"bonus_turn_stamina":                   func(t TechConfig) float64 { return float64(t.Bonus_turn_stamina) },
		"bonus_range":                          func(t TechConfig) float64 { return float64(t.Bonus_range) },
		"splash_damage_percent":                func(t TechConfig) float64 { return t.Splash_damage_percent },
		"splash_max_targets":                   func(t TechConfig) float64 { return float64(t.Splash_max_targets) },
		"reduce_inter_move_cost":               func(t TechConfig) float64 { return float64(t.Reduce_inter_move_cost) },
		"reduce_intra_move_cost":               func(t TechConfig) float64 { return float64(t.Reduce_intra_move_cost) },
		"scale_walls":                          func(t TechConfig) float64 { return boolValue(t.Scale_walls) },
		"required_tech_id":                     func(t TechConfig) float64 { return float64(t.Required_tech_id) },
		"mercenary_cost_markup_delta":          func(t TechConfig) float64 { return t.Mercenary_cost_markup_delta },
		"bonus_gather_capacity":                func(t TechConfig) float64 { return float64(t.Bonus_gather_capacity) },
		"bonus_gather_speed":                   func(t TechConfig) float64 { return float64(t.Bonus_gather_speed) },
		"bonus_resource_capacity":              func(t TechConfig) float64 { return t.Bonus_resource_capacity },
		"renew_cost_discount":                  func(t TechConfig) float64 { return t.Renew_cost_discount },
		"bonus_garrison_capacity":              func(t TechConfig) float64 { return float64(t.Bonus_garrison_capacity) },
		"bonus_garrison_attack_cap_blunt":      func(t TechConfig) float64 { return float64(t.Bonus_garrison_attack_cap_blunt) },
		"bonus_garrison_attack_cap_piercing":   func(t TechConfig) float64 { return float64(t.Bonus_garrison_attack_cap_piercing) },
		"bonus_garrison_attack_cap_magic":      func(t TechConfig) float64 { return float64(t.Bonus_garrison_attack_cap_magic) },
		"bonus_population_support":             func(t TechConfig) float64 { return float64(t.Bonus_population_support) },
		"gather_bonus_bonus_gather_speed":      func(t TechConfig) float64 { return float64(t.Gather_bonus.Bonus_gather_speed) },
		"gather_bonus_bonus_gather_percentage": func(t TechConfig) float64 { return t.Gather_bonus.Bonus_gather_percentage },
	},
	costFields("cost_", func(t TechConfig) RisqResourceCost { return t.Cost }),
	bonusFields("bonus_"),
)
