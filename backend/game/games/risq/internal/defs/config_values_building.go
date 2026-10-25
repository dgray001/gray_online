package defs

func optionalValue(v *int) float64 {
	if v == nil {
		return 0
	}
	return float64(*v)
}

var buildingFields = merge(
	fieldTable[BuildingConfig]{
		"max_health":                   func(b BuildingConfig) float64 { return float64(b.Max_health) },
		"population_support":           func(b BuildingConfig) float64 { return float64(b.Population_support) },
		"garrison_capacity":            func(b BuildingConfig) float64 { return float64(b.Garrison_capacity) },
		"build_stamina":                func(b BuildingConfig) float64 { return float64(b.Build_stamina) },
		"turn_stamina":                 func(b BuildingConfig) float64 { return float64(b.Turn_stamina) },
		"attack_type":                  func(b BuildingConfig) float64 { return float64(b.Attack_type) },
		"attack_blunt":                 func(b BuildingConfig) float64 { return float64(b.Attack_blunt) },
		"attack_piercing":              func(b BuildingConfig) float64 { return float64(b.Attack_piercing) },
		"attack_range":                 func(b BuildingConfig) float64 { return float64(b.Attack_range) },
		"defense_blunt":                func(b BuildingConfig) float64 { return float64(b.Defense_blunt) },
		"defense_piercing":             func(b BuildingConfig) float64 { return float64(b.Defense_piercing) },
		"penetration_blunt":            func(b BuildingConfig) float64 { return float64(b.Penetration_blunt) },
		"penetration_piercing":         func(b BuildingConfig) float64 { return float64(b.Penetration_piercing) },
		"required_tech_id":             func(b BuildingConfig) float64 { return float64(b.Required_tech_id) },
		"max_garrison_attack_blunt":    func(b BuildingConfig) float64 { return optionalValue(b.Max_garrison_attack_blunt) },
		"max_garrison_attack_piercing": func(b BuildingConfig) float64 { return optionalValue(b.Max_garrison_attack_piercing) },
		"max_garrison_attack_magic":    func(b BuildingConfig) float64 { return optionalValue(b.Max_garrison_attack_magic) },
		"max_per_player":               func(b BuildingConfig) float64 { return float64(b.Max_per_player) },
		"is_wall":                      func(b BuildingConfig) float64 { return boolValue(b.Is_wall) },
	},
	costFields("cost_", func(b BuildingConfig) RisqResourceCost { return b.Cost }),
	costFields("gather_renew_cost_", func(b BuildingConfig) RisqResourceCost { return b.Gather.Renew_cost }),
	visionFields(func(b BuildingConfig) RisqVision { return b.Vision }),
	fieldTable[BuildingConfig]{
		"gather_resource_category":  func(b BuildingConfig) float64 { return float64(b.Gather.Resource_category) },
		"gather_speed":              func(b BuildingConfig) float64 { return float64(b.Gather.Base_gather_speed) },
		"gather_starting_resources": func(b BuildingConfig) float64 { return b.Gather.Starting_resources },
		"gather_capacity":           func(b BuildingConfig) float64 { return float64(b.Gather.Gather_capacity) },
		"gather_renew_stamina":      func(b BuildingConfig) float64 { return float64(b.Gather.Renew_stamina) },
	},
)
