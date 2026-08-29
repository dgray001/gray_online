package risq

import (
	_ "embed"
	"encoding/json"
	"fmt"
)

//go:embed config/units.json
var unitsConfigJSON []byte

type UnitConfig struct {
	display_name       string
	max_health         int
	attack_type        AttackType
	attack_blunt       int
	attack_piercing    int
	cost               RisqResourceCost
	production_stamina int
}

type unitConfigJSON struct {
	UnitId            uint32   `json:"unit_id"`
	DisplayName       string   `json:"display_name"`
	MaxHealth         int      `json:"max_health"`
	AttackType        string   `json:"attack_type"`
	AttackBlunt       int      `json:"attack_blunt"`
	AttackPiercing    int      `json:"attack_piercing"`
	Cost              costJSON `json:"cost"`
	ProductionStamina int      `json:"production_stamina"`
}

type costJSON struct {
	Food  float64 `json:"food"`
	Wood  float64 `json:"wood"`
	Stone float64 `json:"stone"`
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
		unitConfigs[e.UnitId] = UnitConfig{
			display_name:    e.DisplayName,
			max_health:      e.MaxHealth,
			attack_type:     attack_type,
			attack_blunt:    e.AttackBlunt,
			attack_piercing: e.AttackPiercing,
			cost: RisqResourceCost{
				food:  e.Cost.Food,
				wood:  e.Cost.Wood,
				stone: e.Cost.Stone,
			},
			production_stamina: e.ProductionStamina,
		}
	}
}
