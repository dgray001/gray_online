package defs

import (
	"encoding/json"
	"fmt"
	"os"
)

type UnitStance uint8

const (
	UnitStance_NONE UnitStance = iota
	UnitStance_PASSIVE
	UnitStance_AGGRESSIVE
	UnitStance_DEFENSIVE
	UnitStance_STAND_GROUND
	UnitStance_END
)

type UnitType uint8

const (
	UnitType_NONE UnitType = iota
	UnitType_ECONOMIC
	UnitType_INFANTRY
	UnitType_ARCHER
	UnitType_CAVALRY
)

func parseUnitType(s string) (UnitType, error) {
	switch s {
	case "economic":
		return UnitType_ECONOMIC, nil
	case "infantry":
		return UnitType_INFANTRY, nil
	case "archer":
		return UnitType_ARCHER, nil
	case "cavalry":
		return UnitType_CAVALRY, nil
	default:
		return UnitType_NONE, fmt.Errorf("unknown unit_type %q", s)
	}
}

type UnitConfig struct {
	Display_name         string
	Description          string
	Unit_type            UnitType
	Max_health           int
	Attack_type          AttackType
	Attack_blunt         int
	Attack_piercing      int
	Attack_range         RisqRange
	Defense_blunt        int
	Defense_piercing     int
	Penetration_blunt    int
	Penetration_piercing int
	Cost                 RisqResourceCost
	Production_stamina   int
	Turn_stamina         int
	Builds               []Producible
	Vision               RisqVision
	Required_tech_id     uint32
}

func (c UnitConfig) CanBuild(building_id uint32) bool {
	for _, p := range c.Builds {
		if p.Kind == ProducibleKind_BUILDING && p.Id == building_id {
			return true
		}
	}
	return false
}

func UnitProductionCost(unit_id uint32) (RisqResourceCost, int) {
	config, ok := UnitConfigs[unit_id]
	if !ok {
		fmt.Fprintln(os.Stderr, "Unknown unit id for production cost: ", unit_id)
		return RisqResourceCost{}, 0
	}
	return config.Cost, config.Production_stamina
}

type unitConfigJSON struct {
	UnitId              uint32           `json:"unit_id"`
	DisplayName         string           `json:"display_name"`
	Description         string           `json:"description"`
	UnitType            string           `json:"unit_type"`
	MaxHealth           int              `json:"max_health"`
	AttackType          string           `json:"attack_type"`
	AttackBlunt         int              `json:"attack_blunt"`
	AttackPiercing      int              `json:"attack_piercing"`
	Range               string           `json:"range"`
	DefenseBlunt        int              `json:"defense_blunt"`
	DefensePiercing     int              `json:"defense_piercing"`
	PenetrationBlunt    int              `json:"penetration_blunt"`
	PenetrationPiercing int              `json:"penetration_piercing"`
	Cost                costJSON         `json:"cost"`
	ProductionStamina   int              `json:"production_stamina"`
	TurnStamina         int              `json:"turn_stamina"`
	Builds              []producibleJSON `json:"builds"`
	Vision              *risqVisionJSON  `json:"vision"`
	RequiredTechId      uint32           `json:"required_tech_id"`
}

type costJSON struct {
	Food  float64 `json:"food"`
	Wood  float64 `json:"wood"`
	Stone float64 `json:"stone"`
	Gold  float64 `json:"gold"`
}

func (c costJSON) toCost() RisqResourceCost {
	return RisqResourceCost{Food: c.Food, Wood: c.Wood, Stone: c.Stone, Gold: c.Gold}
}

type combatBonusJSON struct {
	BonusAttackBlunt         int `json:"bonus_attack_blunt"`
	BonusAttackPiercing      int `json:"bonus_attack_piercing"`
	BonusAttackMagic         int `json:"bonus_attack_magic"`
	BonusDefenseBlunt        int `json:"bonus_defense_blunt"`
	BonusDefensePiercing     int `json:"bonus_defense_piercing"`
	BonusDefenseMagic        int `json:"bonus_defense_magic"`
	BonusPenetrationBlunt    int `json:"bonus_penetration_blunt"`
	BonusPenetrationPiercing int `json:"bonus_penetration_piercing"`
	BonusPenetrationMagic    int `json:"bonus_penetration_magic"`
}

func (j combatBonusJSON) toBonus() CombatBonus {
	return CombatBonus{
		Attack_blunt:         j.BonusAttackBlunt,
		Attack_piercing:      j.BonusAttackPiercing,
		Attack_magic:         j.BonusAttackMagic,
		Defense_blunt:        j.BonusDefenseBlunt,
		Defense_piercing:     j.BonusDefensePiercing,
		Defense_magic:        j.BonusDefenseMagic,
		Penetration_blunt:    j.BonusPenetrationBlunt,
		Penetration_piercing: j.BonusPenetrationPiercing,
		Penetration_magic:    j.BonusPenetrationMagic,
	}
}

func parseAttackType(s string) (AttackType, error) {
	switch s {
	case "", "none":
		return AttackType_NONE, nil
	case "blunt":
		return AttackType_BLUNT, nil
	case "piercing":
		return AttackType_PIERCING, nil
	case "magic":
		return AttackType_MAGIC, nil
	case "blunt_piercing":
		return AttackType_BLUNT_PIERCING, nil
	case "piercing_magic":
		return AttackType_PIERCING_MAGIC, nil
	case "magic_blunt":
		return AttackType_MAGIC_BLUNT, nil
	case "blunt_piercing_magic":
		return AttackType_BLUNT_PIERCING_MAGIC, nil
	default:
		return AttackType_NONE, fmt.Errorf("unknown attack_type %q", s)
	}
}

var UnitConfigs map[uint32]UnitConfig

func loadUnitConfig(data []byte) {
	var entries []unitConfigJSON
	if err := json.Unmarshal(data, &entries); err != nil {
		panic(fmt.Sprintf("failed to parse config/units.json: %v", err))
	}
	UnitConfigs = make(map[uint32]UnitConfig, len(entries))
	for _, e := range entries {
		unit_type, err := parseUnitType(e.UnitType)
		if err != nil {
			panic(fmt.Sprintf("config/units.json unit_id %d: %v", e.UnitId, err))
		}
		attack_type, err := parseAttackType(e.AttackType)
		if err != nil {
			panic(fmt.Sprintf("config/units.json unit_id %d: %v", e.UnitId, err))
		}
		attack_range, err := parseRange(e.Range)
		if err != nil {
			panic(fmt.Sprintf("config/units.json unit_id %d: %v", e.UnitId, err))
		}
		default_kind := ProducibleKind_BUILDING
		builds, err := parseProducibles(e.Builds, &default_kind)
		if err != nil {
			panic(fmt.Sprintf("config/units.json unit_id %d: %v", e.UnitId, err))
		}
		UnitConfigs[e.UnitId] = UnitConfig{
			Display_name:         e.DisplayName,
			Description:          e.Description,
			Unit_type:            unit_type,
			Max_health:           e.MaxHealth,
			Attack_type:          attack_type,
			Attack_blunt:         e.AttackBlunt,
			Attack_piercing:      e.AttackPiercing,
			Attack_range:         attack_range,
			Defense_blunt:        e.DefenseBlunt,
			Defense_piercing:     e.DefensePiercing,
			Penetration_blunt:    e.PenetrationBlunt,
			Penetration_piercing: e.PenetrationPiercing,
			Cost:                 e.Cost.toCost(),
			Production_stamina:   e.ProductionStamina,
			Turn_stamina:         e.TurnStamina,
			Builds:               builds,
			Vision:               resolveVision(e.Vision),
			Required_tech_id:     e.RequiredTechId,
		}
	}
}
