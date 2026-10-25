package defs

var unitFields = merge(
	fieldTable[UnitConfig]{
		"unit_type":            func(u UnitConfig) float64 { return float64(u.Unit_type) },
		"max_health":           func(u UnitConfig) float64 { return float64(u.Max_health) },
		"attack_type":          func(u UnitConfig) float64 { return float64(u.Attack_type) },
		"attack_blunt":         func(u UnitConfig) float64 { return float64(u.Attack_blunt) },
		"attack_piercing":      func(u UnitConfig) float64 { return float64(u.Attack_piercing) },
		"attack_range":         func(u UnitConfig) float64 { return float64(u.Attack_range) },
		"defense_blunt":        func(u UnitConfig) float64 { return float64(u.Defense_blunt) },
		"defense_piercing":     func(u UnitConfig) float64 { return float64(u.Defense_piercing) },
		"penetration_blunt":    func(u UnitConfig) float64 { return float64(u.Penetration_blunt) },
		"penetration_piercing": func(u UnitConfig) float64 { return float64(u.Penetration_piercing) },
		"production_stamina":   func(u UnitConfig) float64 { return float64(u.Production_stamina) },
		"turn_stamina":         func(u UnitConfig) float64 { return float64(u.Turn_stamina) },
		"required_tech_id":     func(u UnitConfig) float64 { return float64(u.Required_tech_id) },
	},
	costFields("cost_", func(u UnitConfig) RisqResourceCost { return u.Cost }),
	visionFields(func(u UnitConfig) RisqVision { return u.Vision }),
)
