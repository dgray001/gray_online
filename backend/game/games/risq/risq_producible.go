package risq

import (
	"fmt"

	"github.com/gin-gonic/gin"
)

type ProducibleKind uint8

const (
	ProducibleKind_NONE ProducibleKind = iota
	ProducibleKind_UNIT
	ProducibleKind_TECH
	ProducibleKind_BUILDING
)

func parseProducibleKind(s string) (ProducibleKind, error) {
	switch s {
	case "unit":
		return ProducibleKind_UNIT, nil
	case "tech":
		return ProducibleKind_TECH, nil
	case "building":
		return ProducibleKind_BUILDING, nil
	default:
		return ProducibleKind_NONE, fmt.Errorf("unknown producible kind %q", s)
	}
}

type Producible struct {
	row  int
	col  int
	kind ProducibleKind
	id   uint32
}

type producibleJSON struct {
	Row  int    `json:"row"`
	Col  int    `json:"col"`
	Kind string `json:"kind"`
	Id   uint32 `json:"id"`
}

func parseProducibles(raw []producibleJSON, default_kind *ProducibleKind) ([]Producible, error) {
	producibles := make([]Producible, 0, len(raw))
	for _, p := range raw {
		var kind ProducibleKind
		var err error
		if p.Kind == "" && default_kind != nil {
			kind = *default_kind
		} else {
			kind, err = parseProducibleKind(p.Kind)
		}
		if err != nil {
			return nil, err
		}
		producibles = append(producibles, Producible{row: p.Row, col: p.Col, kind: kind, id: p.Id})
	}
	return producibles, nil
}

func (p Producible) toFrontend() gin.H {
	entry := gin.H{
		"row":  p.row,
		"col":  p.col,
		"kind": p.kind,
		"id":   p.id,
	}
	switch p.kind {
	case ProducibleKind_UNIT:
		unit := unitConfigs[p.id]
		cost, stamina_cost := unitProductionCost(p.id)
		entry["cost"] = cost.toFrontend()
		entry["stamina_cost"] = stamina_cost
		entry["display_name"] = unit.display_name
		entry["description"] = unit.description
		entry["required_tech_id"] = unit.required_tech_id
		entry["stats"] = gin.H{
			"health":               unit.max_health,
			"attack_type":          unit.attack_type,
			"attack_blunt":         unit.attack_blunt,
			"attack_piercing":      unit.attack_piercing,
			"attack_range":         unit.attack_range,
			"defense_blunt":        unit.defense_blunt,
			"defense_piercing":     unit.defense_piercing,
			"penetration_blunt":    unit.penetration_blunt,
			"penetration_piercing": unit.penetration_piercing,
		}
	case ProducibleKind_BUILDING:
		cost, stamina_cost := buildingProductionCost(p.id)
		entry["cost"] = cost.toFrontend()
		entry["stamina_cost"] = stamina_cost
		entry["display_name"] = buildingConfigs[p.id].display_name
		entry["description"] = buildingConfigs[p.id].description
		entry["required_tech_id"] = buildingConfigs[p.id].required_tech_id
	case ProducibleKind_TECH:
		tech := techConfigs[p.id]
		entry["cost"] = tech.cost.toFrontend()
		entry["stamina_cost"] = tech.research_stamina
		entry["display_name"] = tech.display_name
		entry["description"] = tech.description
		entry["required_tech_id"] = tech.required_tech_id
		affects_unit_types := make([]int, len(tech.affects_unit_types))
		for i, unit_type := range tech.affects_unit_types {
			affects_unit_types[i] = int(unit_type)
		}
		entry["affects_unit_ids"] = tech.affects_unit_ids
		entry["affects_unit_types"] = affects_unit_types
		bonus := tech.bonus.toFrontend()
		bonus["max_health"] = tech.bonus_max_health
		bonus["turn_stamina"] = tech.bonus_turn_stamina
		entry["bonus"] = bonus
	}
	return entry
}
