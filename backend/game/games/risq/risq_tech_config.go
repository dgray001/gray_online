package risq

import (
	_ "embed"
	"encoding/json"
	"fmt"
)

//go:embed config/techs.json
var techsConfigJSON []byte

type TechConfig struct {
	display_name          string
	description           string
	cost                  RisqResourceCost
	research_stamina      int
	affects_unit_ids      []uint32
	affects_unit_types    []UnitType
	target_unit_ids       []uint32
	target_types          []TargetType
	bonus_max_health      int
	bonus_turn_stamina    int
	bonus                 CombatBonus
	required_tech_id      uint32
	unlocks_mercenary_ids []uint32
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

var techConfigs map[uint32]TechConfig

func init() {
	var entries []techConfigJSON
	if err := json.Unmarshal(techsConfigJSON, &entries); err != nil {
		panic(fmt.Sprintf("failed to parse config/techs.json: %v", err))
	}
	techConfigs = make(map[uint32]TechConfig, len(entries))
	for _, e := range entries {
		techConfigs[e.TechId] = parseTechConfigEntry(e, "config/techs.json tech_id")
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
		display_name:          e.DisplayName,
		description:           e.Description,
		cost:                  e.Cost.toCost(),
		research_stamina:      e.ResearchStamina,
		affects_unit_ids:      e.AffectsUnitIds,
		affects_unit_types:    affects_unit_types,
		target_unit_ids:       e.TargetUnitIds,
		target_types:          target_types,
		bonus_max_health:      e.BonusMaxHealth,
		bonus_turn_stamina:    e.BonusTurnStamina,
		bonus:                 e.combatBonusJSON.toBonus(),
		required_tech_id:      e.RequiredTechId,
		unlocks_mercenary_ids: e.UnlocksMercenaryIds,
	}
}

func (tech TechConfig) appliesTo(unit_id uint32, unit_type UnitType) bool {
	for _, id := range tech.affects_unit_ids {
		if id == unit_id {
			return true
		}
	}
	for _, t := range tech.affects_unit_types {
		if t == unit_type {
			return true
		}
	}
	return false
}

func (tech TechConfig) hasTargetFilter() bool {
	return len(tech.target_unit_ids) > 0 || len(tech.target_types) > 0
}

func (tech TechConfig) matchesTarget(target_unit_id uint32, target_unit_type UnitType, target_orderable_type OrderableType) bool {
	if target_orderable_type == OrderableType_UNIT {
		for _, id := range tech.target_unit_ids {
			if id == target_unit_id {
				return true
			}
		}
	}
	actual := targetTypeOf(target_orderable_type, target_unit_type)
	for _, tt := range tech.target_types {
		if tt.matches(actual) {
			return true
		}
	}
	return false
}

func applyTechBonus(u *RisqUnit, tech TechConfig) {
	u.turn_stamina += tech.bonus_turn_stamina
	u.cs.setMaxHealth(u.cs.max_health + tech.bonus_max_health)
	if tech.hasTargetFilter() {
		return
	}
	tech.bonus.applyAll(&u.cs)
}

func sumTargetedBonus(unit_id uint32, unit_type UnitType, tech_ids []uint32, target_unit_id uint32, target_unit_type UnitType, target_orderable_type OrderableType) CombatBonus {
	var sum CombatBonus
	add := func(tech TechConfig) {
		if !tech.hasTargetFilter() || !tech.appliesTo(unit_id, unit_type) || !tech.matchesTarget(target_unit_id, target_unit_type, target_orderable_type) {
			return
		}
		sum = sum.add(tech.bonus)
	}
	for _, bonus := range bonusConfigs {
		add(bonus)
	}
	for _, id := range tech_ids {
		if tech, ok := techConfigs[id]; ok {
			add(tech)
		}
	}
	return sum
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
	tech, ok := techConfigs[tech_id]
	if !ok {
		return
	}
	for _, unit := range player.units {
		if tech.appliesTo(unit.unit_id, unit.unitType()) {
			applyTechBonus(unit, tech)
		}
	}
	for _, unit_id := range tech.unlocks_mercenary_ids {
		player.available_mercenaries[unit_id] = true
	}
}
