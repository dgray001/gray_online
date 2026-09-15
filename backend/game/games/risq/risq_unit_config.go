package risq

import (
	_ "embed"
	"encoding/json"
	"fmt"
)

//go:embed config/units.json
var unitsConfigJSON []byte

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
	display_name         string
	description          string
	unit_type            UnitType
	max_health           int
	attack_type          AttackType
	attack_blunt         int
	attack_piercing      int
	defense_blunt        int
	defense_piercing     int
	penetration_blunt    int
	penetration_piercing int
	cost                 RisqResourceCost
	production_stamina   int
	turn_stamina         int
	builds               []Producible
	vision               RisqVision
	required_tech_id     uint32
}

func (c UnitConfig) canBuild(building_id uint32) bool {
	for _, p := range c.builds {
		if p.kind == ProducibleKind_BUILDING && p.id == building_id {
			return true
		}
	}
	return false
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

var unitConfigs map[uint32]UnitConfig

func init() {
	var entries []unitConfigJSON
	if err := json.Unmarshal(unitsConfigJSON, &entries); err != nil {
		panic(fmt.Sprintf("failed to parse config/units.json: %v", err))
	}
	unitConfigs = make(map[uint32]UnitConfig, len(entries))
	for _, e := range entries {
		unit_type, err := parseUnitType(e.UnitType)
		if err != nil {
			panic(fmt.Sprintf("config/units.json unit_id %d: %v", e.UnitId, err))
		}
		attack_type, err := parseAttackType(e.AttackType)
		if err != nil {
			panic(fmt.Sprintf("config/units.json unit_id %d: %v", e.UnitId, err))
		}
		default_kind := ProducibleKind_BUILDING
		builds, err := parseProducibles(e.Builds, &default_kind)
		if err != nil {
			panic(fmt.Sprintf("config/units.json unit_id %d: %v", e.UnitId, err))
		}
		unitConfigs[e.UnitId] = UnitConfig{
			display_name:         e.DisplayName,
			description:          e.Description,
			unit_type:            unit_type,
			max_health:           e.MaxHealth,
			attack_type:          attack_type,
			attack_blunt:         e.AttackBlunt,
			attack_piercing:      e.AttackPiercing,
			defense_blunt:        e.DefenseBlunt,
			defense_piercing:     e.DefensePiercing,
			penetration_blunt:    e.PenetrationBlunt,
			penetration_piercing: e.PenetrationPiercing,
			cost: RisqResourceCost{
				food:  e.Cost.Food,
				wood:  e.Cost.Wood,
				stone: e.Cost.Stone,
				gold:  e.Cost.Gold,
			},
			production_stamina: e.ProductionStamina,
			turn_stamina:       e.TurnStamina,
			builds:             builds,
			vision:             resolveVision(e.Vision),
			required_tech_id:   e.RequiredTechId,
		}
	}
}
