package defs

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
	Row  int
	Col  int
	Kind ProducibleKind
	Id   uint32
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
		producibles = append(producibles, Producible{Row: p.Row, Col: p.Col, Kind: kind, Id: p.Id})
	}
	return producibles, nil
}

func (p Producible) ToFrontend(c *Config) gin.H {
	entry := gin.H{
		"row":  p.Row,
		"col":  p.Col,
		"kind": p.Kind,
		"id":   p.Id,
	}
	switch p.Kind {
	case ProducibleKind_UNIT:
		unit := c.Units[p.Id]
		cost, stamina_cost := c.UnitProductionCost(p.Id)
		entry["cost"] = cost.ToFrontend()
		entry["stamina_cost"] = stamina_cost
		entry["display_name"] = unit.Display_name
		entry["description"] = unit.Description
		entry["unit_type"] = unit.Unit_type
		entry["required_tech_id"] = unit.Required_tech_id
		entry["stats"] = gin.H{
			"health":               unit.Max_health,
			"attack_type":          unit.Attack_type,
			"attack_blunt":         unit.Attack_blunt,
			"attack_piercing":      unit.Attack_piercing,
			"attack_range":         unit.Attack_range,
			"defense_blunt":        unit.Defense_blunt,
			"defense_piercing":     unit.Defense_piercing,
			"penetration_blunt":    unit.Penetration_blunt,
			"penetration_piercing": unit.Penetration_piercing,
		}
	case ProducibleKind_BUILDING:
		building := c.Buildings[p.Id]
		cost, stamina_cost := c.BuildingProductionCost(p.Id)
		entry["cost"] = cost.ToFrontend()
		entry["stamina_cost"] = stamina_cost
		entry["display_name"] = building.Display_name
		entry["description"] = building.Description
		entry["required_tech_id"] = building.Required_tech_id
		entry["population_support"] = building.Population_support
		entry["garrison_capacity"] = building.Garrison_capacity
		entry["max_per_player"] = building.Max_per_player
	case ProducibleKind_TECH:
		tech := c.Techs[p.Id]
		entry["cost"] = tech.Cost.ToFrontend()
		entry["stamina_cost"] = tech.Research_stamina
		entry["display_name"] = tech.Display_name
		entry["description"] = tech.Description
		entry["required_tech_id"] = tech.Required_tech_id
		affects_unit_types := make([]int, len(tech.Affects_unit_types))
		for i, unit_type := range tech.Affects_unit_types {
			affects_unit_types[i] = int(unit_type)
		}
		entry["affects_unit_ids"] = tech.Affects_unit_ids
		entry["affects_unit_types"] = affects_unit_types
		bonus := tech.Bonus.ToFrontend()
		bonus["max_health"] = tech.Bonus_max_health
		bonus["turn_stamina"] = tech.Bonus_turn_stamina
		entry["bonus"] = bonus
		unlocks_mercenaries := make([]gin.H, len(tech.Unlocks_mercenary_ids))
		for i, unit_id := range tech.Unlocks_mercenary_ids {
			unlocks_mercenaries[i] = gin.H{"id": unit_id, "display_name": c.Units[unit_id].Display_name}
		}
		entry["unlocks_mercenaries"] = unlocks_mercenaries
	}
	return entry
}
