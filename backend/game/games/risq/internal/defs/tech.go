package defs

import (
	"encoding/json"
	"fmt"
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
}

type techConfigJSON struct {
	TechId           uint32   `json:"tech_id"`
	DisplayName      string   `json:"display_name"`
	Description      string   `json:"description"`
	Cost             costJSON `json:"cost"`
	ResearchStamina  int      `json:"research_stamina"`
	AffectsUnitIds   []uint32 `json:"affects_unit_ids"`
	AffectsUnitTypes []string `json:"affects_unit_types"`
	TargetUnitIds    []uint32 `json:"target_unit_ids"`
	TargetTypes      []string `json:"target_types"`
	BonusMaxHealth   int      `json:"bonus_max_health"`
	BonusTurnStamina int      `json:"bonus_turn_stamina"`
	combatBonusJSON
	RequiredTechId      uint32   `json:"required_tech_id"`
	UnlocksMercenaryIds []uint32 `json:"unlocks_mercenary_ids"`
}

var TechConfigs map[uint32]TechConfig

func loadTechConfig(data []byte) {
	var entries []techConfigJSON
	if err := json.Unmarshal(data, &entries); err != nil {
		panic(fmt.Sprintf("failed to parse config/techs.json: %v", err))
	}
	TechConfigs = make(map[uint32]TechConfig, len(entries))
	for _, e := range entries {
		TechConfigs[e.TechId] = parseTechConfigEntry(e, "config/techs.json tech_id")
	}
}

func parseTechConfigEntry(e techConfigJSON, source string) TechConfig {
	if len(e.AffectsUnitIds) == 0 && len(e.AffectsUnitTypes) == 0 && len(e.UnlocksMercenaryIds) == 0 {
		panic(fmt.Sprintf("%s %d: must specify affects_unit_ids, affects_unit_types, or unlocks_mercenary_ids", source, e.TechId))
	}
	affects_unit_types := make([]UnitType, len(e.AffectsUnitTypes))
	for i, s := range e.AffectsUnitTypes {
		unit_type, err := parseUnitType(s)
		if err != nil {
			panic(fmt.Sprintf("%s %d: %v", source, e.TechId, err))
		}
		affects_unit_types[i] = unit_type
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
	}
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

func SumTargetedBonus(unit_id uint32, unit_type UnitType, tech_ids []uint32, target_unit_id uint32, target_unit_type UnitType, target_orderable_type OrderableType) CombatBonus {
	var sum CombatBonus
	add := func(tech TechConfig) {
		if !tech.HasTargetFilter() || !tech.AppliesTo(unit_id, unit_type) || !tech.MatchesTarget(target_unit_id, target_unit_type, target_orderable_type) {
			return
		}
		sum = sum.Add(tech.Bonus)
	}
	for _, bonus := range BonusConfigs {
		add(bonus)
	}
	for _, id := range tech_ids {
		if tech, ok := TechConfigs[id]; ok {
			add(tech)
		}
	}
	return sum
}

var BonusConfigs []TechConfig

func loadBonusConfig(data []byte) {
	var entries []techConfigJSON
	if err := json.Unmarshal(data, &entries); err != nil {
		panic(fmt.Sprintf("failed to parse config/bonuses.json: %v", err))
	}
	BonusConfigs = make([]TechConfig, len(entries))
	for i, e := range entries {
		BonusConfigs[i] = parseTechConfigEntry(e, "config/bonuses.json tech_id")
	}
}
