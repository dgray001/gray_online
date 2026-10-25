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
		"research_stamina":            func(t TechConfig) float64 { return float64(t.Research_stamina) },
		"bonus_max_health":            func(t TechConfig) float64 { return float64(t.Bonus_max_health) },
		"bonus_turn_stamina":          func(t TechConfig) float64 { return float64(t.Bonus_turn_stamina) },
		"required_tech_id":            func(t TechConfig) float64 { return float64(t.Required_tech_id) },
		"mercenary_cost_markup_delta": func(t TechConfig) float64 { return t.Mercenary_cost_markup_delta },
	},
	costFields("cost_", func(t TechConfig) RisqResourceCost { return t.Cost }),
	bonusFields("bonus_"),
)
