package risq

import (
	_ "embed"
	"encoding/json"
	"fmt"
)

//go:embed config/units.json
var unitsConfigJSON []byte

type UnitConfig struct {
	display_name         string
	description          string
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
}

type costJSON struct {
	Food  float64 `json:"food"`
	Wood  float64 `json:"wood"`
	Stone float64 `json:"stone"`
	Gold  float64 `json:"gold"`
}

func parseAttackType(s string) (AttackType, error) {
	switch s {
	case "", "NONE":
		return AttackType_NONE, nil
	case "BLUNT":
		return AttackType_BLUNT, nil
	case "PIERCING":
		return AttackType_PIERCING, nil
	case "MAGIC":
		return AttackType_MAGIC, nil
	case "BLUNT_PIERCING":
		return AttackType_BLUNT_PIERCING, nil
	case "PIERCING_MAGIC":
		return AttackType_PIERCING_MAGIC, nil
	case "MAGIC_BLUNT":
		return AttackType_MAGIC_BLUNT, nil
	case "BLUNT_PIERCING_MAGIC":
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
		}
	}
}
