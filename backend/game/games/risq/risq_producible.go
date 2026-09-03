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
		cost, stamina_cost := unitProductionCost(p.id)
		entry["cost"] = cost.toFrontend()
		entry["stamina_cost"] = stamina_cost
		entry["display_name"] = unitConfigs[p.id].display_name
		entry["description"] = unitConfigs[p.id].description
	case ProducibleKind_BUILDING:
		cost, stamina_cost := buildingProductionCost(p.id)
		entry["cost"] = cost.toFrontend()
		entry["stamina_cost"] = stamina_cost
		entry["display_name"] = buildingConfigs[p.id].display_name
		entry["description"] = buildingConfigs[p.id].description
	}
	return entry
}
