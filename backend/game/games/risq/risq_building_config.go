package risq

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"os"

	"github.com/gin-gonic/gin"
)

//go:embed config/buildings.json
var buildingsConfigJSON []byte

type ProducibleKind uint8

const (
	ProducibleKind_UNIT ProducibleKind = iota
	ProducibleKind_TECH
)

func parseProducibleKind(s string) (ProducibleKind, error) {
	switch s {
	case "unit":
		return ProducibleKind_UNIT, nil
	case "tech":
		return ProducibleKind_TECH, nil
	default:
		return ProducibleKind_UNIT, fmt.Errorf("unknown producible kind %q", s)
	}
}

type Producible struct {
	row  int
	col  int
	kind ProducibleKind
	id   uint32
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
		cost, _ := unitProductionCost(p.id)
		entry["cost"] = cost.toFrontend()
		entry["display_name"] = unitConfigs[p.id].display_name
	}
	return entry
}

type BuildingConfig struct {
	display_name       string
	max_health         int
	population_support uint16
	produces           []Producible
	// zero-value cost/build_stamina means this building type can't be constructed by a unit
	cost          RisqResourceCost
	build_stamina int
}

func (c BuildingConfig) canProduce(unit_id uint32) bool {
	for _, p := range c.produces {
		if p.kind == ProducibleKind_UNIT && p.id == unit_id {
			return true
		}
	}
	return false
}

// Returns the resource cost and total build stamina required for a unit to construct this building type
func buildingProductionCost(building_id uint32) (RisqResourceCost, int) {
	config, ok := buildingConfigs[building_id]
	if !ok || config.build_stamina <= 0 {
		fmt.Fprintln(os.Stderr, "Unknown or unbuildable building id: ", building_id)
		return RisqResourceCost{}, 0
	}
	return config.cost, config.build_stamina
}

type producibleJSON struct {
	Row  int    `json:"row"`
	Col  int    `json:"col"`
	Kind string `json:"kind"`
	Id   uint32 `json:"id"`
}

type buildingConfigJSON struct {
	BuildingId        uint32           `json:"building_id"`
	DisplayName       string           `json:"display_name"`
	MaxHealth         int              `json:"max_health"`
	PopulationSupport uint16           `json:"population_support"`
	Produces          []producibleJSON `json:"produces"`
	Cost              costJSON         `json:"cost"`
	BuildStamina      int              `json:"build_stamina"`
}

var buildingConfigs map[uint32]BuildingConfig

func init() {
	var entries []buildingConfigJSON
	if err := json.Unmarshal(buildingsConfigJSON, &entries); err != nil {
		panic(fmt.Sprintf("failed to parse config/buildings.json: %v", err))
	}
	buildingConfigs = make(map[uint32]BuildingConfig, len(entries))
	for _, e := range entries {
		produces := make([]Producible, 0, len(e.Produces))
		for _, p := range e.Produces {
			kind, err := parseProducibleKind(p.Kind)
			if err != nil {
				panic(fmt.Sprintf("config/buildings.json building_id %d: %v", e.BuildingId, err))
			}
			produces = append(produces, Producible{
				row:  p.Row,
				col:  p.Col,
				kind: kind,
				id:   p.Id,
			})
		}
		buildingConfigs[e.BuildingId] = BuildingConfig{
			display_name:       e.DisplayName,
			max_health:         e.MaxHealth,
			population_support: e.PopulationSupport,
			produces:           produces,
			cost: RisqResourceCost{
				food:  e.Cost.Food,
				wood:  e.Cost.Wood,
				stone: e.Cost.Stone,
			},
			build_stamina: e.BuildStamina,
		}
	}
}
