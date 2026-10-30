package defs

import (
	"encoding/json"
	"fmt"
	"slices"
)

type TechConfig struct {
	Display_name          string
	Description           string
	Cost                  RisqResourceCost
	Research_stamina      int
	Affects_unit_ids      []uint32
	Affects_unit_types    []UnitType
	Target_unit_ids       []uint32
	Target_types          []TargetType
	Bonus_max_health      int
	Bonus_turn_stamina    int
	Bonus                 CombatBonus
	Required_tech_id      uint32
	Unlocks_mercenary_ids []uint32
	// added to the 1.3 mercenary markup; deltas from every bonus and researched tech stack
	Mercenary_cost_markup_delta        float64
	Affects_building_ids               []uint32
	Bonus_gather_capacity              int
	Bonus_gather_speed                 int
	Bonus_resource_capacity            float64
	Renew_cost_discount                float64
	Bonus_garrison_capacity            int
	Bonus_garrison_attack_cap_blunt    int
	Bonus_garrison_attack_cap_piercing int
	Bonus_garrison_attack_cap_magic    int
	Bonus_population_support           int
	Gather_bonus                       GatherBonus
	Bonus_range                        int
	Splash_damage_percent              float64
	Splash_max_targets                 int
	Reduce_inter_move_cost             int
	Reduce_intra_move_cost             int
	Scale_walls                        bool
	Additional_effects                 []TechConfig
}

type techConfigJSON struct {
	TechId            uint32            `json:"tech_id"`
	DisplayName       string            `json:"display_name"`
	Description       string            `json:"description"`
	Cost              costJSON          `json:"cost"`
	AdditionalEffects []json.RawMessage `json:"additional_effects"`
	techEffectJSON
}

type techEffectJSON struct {
	ResearchStamina     int      `json:"research_stamina"`
	AffectsUnitIds      []uint32 `json:"affects_unit_ids"`
	AffectsUnitTypes    []string `json:"affects_unit_types"`
	TargetUnitIds       []uint32 `json:"target_unit_ids"`
	TargetTypes         []string `json:"target_types"`
	BonusMaxHealth      int      `json:"bonus_max_health"`
	BonusTurnStamina    int      `json:"bonus_turn_stamina"`
	BonusRange          int      `json:"bonus_range"`
	SplashDamagePercent float64  `json:"splash_damage_percent"`
	SplashMaxTargets    int      `json:"splash_max_targets"`
	ReduceInterMoveCost int      `json:"reduce_inter_move_cost"`
	ReduceIntraMoveCost int      `json:"reduce_intra_move_cost"`
	ScaleWalls          bool     `json:"scale_walls"`
	combatBonusJSON
	RequiredTechId                 uint32          `json:"required_tech_id"`
	UnlocksMercenaryIds            []uint32        `json:"unlocks_mercenary_ids"`
	MercenaryCostMarkupDelta       float64         `json:"mercenary_cost_markup_delta"`
	AffectsBuildingIds             []uint32        `json:"affects_building_ids"`
	BonusGatherCapacity            int             `json:"bonus_gather_capacity"`
	BonusGatherSpeed               int             `json:"bonus_gather_speed"`
	BonusResourceCapacity          float64         `json:"bonus_resource_capacity"`
	RenewCostDiscount              float64         `json:"renew_cost_discount"`
	BonusGarrisonCapacity          int             `json:"bonus_garrison_capacity"`
	BonusGarrisonAttackCapBlunt    int             `json:"bonus_garrison_attack_cap_blunt"`
	BonusGarrisonAttackCapPiercing int             `json:"bonus_garrison_attack_cap_piercing"`
	BonusGarrisonAttackCapMagic    int             `json:"bonus_garrison_attack_cap_magic"`
	BonusPopulationSupport         int             `json:"bonus_population_support"`
	GatherBonus                    gatherBonusJSON `json:"gather_bonus"`
}

var TechConfigs map[uint32]TechConfig

func (c *Config) loadTechs(data []byte) {
	var entries []techConfigJSON
	if err := json.Unmarshal(data, &entries); err != nil {
		panic(fmt.Sprintf("failed to parse config/techs.json: %v", err))
	}
	c.Techs = make(map[uint32]TechConfig, len(entries))
	for _, e := range entries {
		c.Techs[e.TechId] = parseTechConfigEntry(e, "config/techs.json tech_id")
	}
}

func parseTechConfigEntry(e techConfigJSON, source string) TechConfig {
	additional_effects := parseAdditionalTechEffects(e.AdditionalEffects, source, e.TechId)
	if len(additional_effects) == 0 && len(e.AffectsUnitIds) == 0 && len(e.AffectsUnitTypes) == 0 && len(e.AffectsBuildingIds) == 0 && len(e.UnlocksMercenaryIds) == 0 && e.MercenaryCostMarkupDelta == 0 {
		panic(fmt.Sprintf("%s %d: must specify affects_unit_ids, affects_unit_types, affects_building_ids, unlocks_mercenary_ids, or mercenary_cost_markup_delta", source, e.TechId))
	}
	if e.RenewCostDiscount < 0 || e.RenewCostDiscount > 1 {
		panic(fmt.Sprintf("%s %d: renew_cost_discount must be in [0, 1]", source, e.TechId))
	}
	if e.BonusRange < 0 || e.BonusRange > int(RisqRange_SECONDARY-RisqRange_ZONE) {
		panic(fmt.Sprintf("%s %d: bonus_range must be between 0 and %d", source, e.TechId, RisqRange_SECONDARY-RisqRange_ZONE))
	}
	affects_unit_types := make([]UnitType, len(e.AffectsUnitTypes))
	for i, s := range e.AffectsUnitTypes {
		unit_type, err := parseUnitType(s)
		if err != nil {
			panic(fmt.Sprintf("%s %d: %v", source, e.TechId, err))
		}
		affects_unit_types[i] = unit_type
	}
	gather_bonus, err := parseGatherBonus(e.GatherBonus)
	if err != nil {
		panic(fmt.Sprintf("%s %d: %v", source, e.TechId, err))
	}
	if gather_bonus.IsSet() && len(e.AffectsUnitIds) == 0 && len(e.AffectsUnitTypes) == 0 {
		panic(fmt.Sprintf("%s %d: gather_bonus needs affects_unit_ids or affects_unit_types", source, e.TechId))
	}
	target_types := make([]TargetType, len(e.TargetTypes))
	for i, s := range e.TargetTypes {
		target_type, err := parseTargetType(s)
		if err != nil {
			panic(fmt.Sprintf("%s %d: %v", source, e.TechId, err))
		}
		target_types[i] = target_type
	}
	return TechConfig{
		Display_name:          e.DisplayName,
		Description:           e.Description,
		Cost:                  e.Cost.toCost(),
		Research_stamina:      e.ResearchStamina,
		Affects_unit_ids:      e.AffectsUnitIds,
		Affects_unit_types:    affects_unit_types,
		Target_unit_ids:       e.TargetUnitIds,
		Target_types:          target_types,
		Bonus_max_health:      e.BonusMaxHealth,
		Bonus_turn_stamina:    e.BonusTurnStamina,
		Bonus:                 e.combatBonusJSON.toBonus(),
		Required_tech_id:      e.RequiredTechId,
		Unlocks_mercenary_ids: e.UnlocksMercenaryIds,

		Mercenary_cost_markup_delta:        e.MercenaryCostMarkupDelta,
		Affects_building_ids:               e.AffectsBuildingIds,
		Bonus_gather_capacity:              e.BonusGatherCapacity,
		Bonus_gather_speed:                 e.BonusGatherSpeed,
		Bonus_resource_capacity:            e.BonusResourceCapacity,
		Renew_cost_discount:                e.RenewCostDiscount,
		Bonus_garrison_capacity:            e.BonusGarrisonCapacity,
		Bonus_garrison_attack_cap_blunt:    e.BonusGarrisonAttackCapBlunt,
		Bonus_garrison_attack_cap_piercing: e.BonusGarrisonAttackCapPiercing,
		Bonus_garrison_attack_cap_magic:    e.BonusGarrisonAttackCapMagic,
		Bonus_population_support:           e.BonusPopulationSupport,
		Gather_bonus:                       gather_bonus,
		Bonus_range:                        e.BonusRange,
		Splash_damage_percent:              e.SplashDamagePercent,
		Splash_max_targets:                 e.SplashMaxTargets,
		Reduce_inter_move_cost:             e.ReduceInterMoveCost,
		Reduce_intra_move_cost:             e.ReduceIntraMoveCost,
		Scale_walls:                        e.ScaleWalls,
		Additional_effects:                 additional_effects,
	}
}

func (tech TechConfig) AppliesToBuilding(building_id uint32) bool {
	return slices.Contains(tech.Affects_building_ids, building_id)
}

func (c *Config) RenewCostMultiplier(building_id uint32, tech_ids []uint32) float64 {
	multiplier := 1.0
	for _, tech := range c.ActiveTechEffects(tech_ids) {
		if tech.AppliesToBuilding(building_id) {
			multiplier *= 1 - tech.Renew_cost_discount
		}
	}
	return multiplier
}

func (tech TechConfig) AppliesTo(unit_id uint32, unit_type UnitType) bool {
	for _, id := range tech.Affects_unit_ids {
		if id == unit_id {
			return true
		}
	}
	for _, t := range tech.Affects_unit_types {
		if t == unit_type {
			return true
		}
	}
	return false
}

func (tech TechConfig) HasTargetFilter() bool {
	return len(tech.Target_unit_ids) > 0 || len(tech.Target_types) > 0
}

func (tech TechConfig) MatchesTarget(target_unit_id uint32, target_unit_type UnitType, target_orderable_type OrderableType) bool {
	if target_orderable_type == OrderableType_UNIT {
		for _, id := range tech.Target_unit_ids {
			if id == target_unit_id {
				return true
			}
		}
	}
	actual := targetTypeOf(target_orderable_type, target_unit_type)
	for _, tt := range tech.Target_types {
		if tt.Matches(actual) {
			return true
		}
	}
	return false
}

// Uses the default config, for callers without a game config
func SumTargetedBonus(unit_id uint32, unit_type UnitType, tech_ids []uint32, target_unit_id uint32, target_unit_type UnitType, target_orderable_type OrderableType) CombatBonus {
	return DefaultConfig().SumTargetedBonus(unit_id, unit_type, tech_ids, target_unit_id, target_unit_type, target_orderable_type)
}

func (c *Config) SumTargetedBonus(unit_id uint32, unit_type UnitType, tech_ids []uint32, target_unit_id uint32, target_unit_type UnitType, target_orderable_type OrderableType) CombatBonus {
	var sum CombatBonus
	add := func(tech TechConfig) {
		if !tech.HasTargetFilter() || !tech.AppliesTo(unit_id, unit_type) || !tech.MatchesTarget(target_unit_id, target_unit_type, target_orderable_type) {
			return
		}
		sum = sum.Add(tech.Bonus)
	}
	for _, effect := range c.ActiveTechEffects(tech_ids) {
		add(effect)
	}
	return sum
}

// Every bonus and each researched tech adds its markup delta, so the total never depends on order
func (c *Config) MercenaryCostMarkupDelta(tech_ids []uint32) float64 {
	delta := 0.0
	for _, effect := range c.ActiveTechEffects(tech_ids) {
		delta += effect.Mercenary_cost_markup_delta
	}
	return delta
}

var BonusConfigs []TechConfig

func (c *Config) loadBonuses(data []byte) {
	var entries []techConfigJSON
	if err := json.Unmarshal(data, &entries); err != nil {
		panic(fmt.Sprintf("failed to parse config/bonuses.json: %v", err))
	}
	bonuses := make([]TechConfig, len(entries))
	for i, e := range entries {
		bonuses[i] = parseTechConfigEntry(e, "config/bonuses.json tech_id")
	}
	*c.bonuses = bonuses
}
