package risq

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"os"
)

//go:embed config/buildings.json
var buildingsConfigJSON []byte

type BuildingConfig struct {
	display_name       string
	max_health         int
	population_support uint16
	produces           []uint32
	// zero-value cost/build_stamina means this building type can't be constructed by a unit
	cost          RisqResourceCost
	build_stamina int
}

func (c BuildingConfig) canProduce(unit_id uint32) bool {
	for _, id := range c.produces {
		if id == unit_id {
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

type buildingConfigJSON struct {
	BuildingId        uint32   `json:"building_id"`
	DisplayName       string   `json:"display_name"`
	MaxHealth         int      `json:"max_health"`
	PopulationSupport uint16   `json:"population_support"`
	Produces          []uint32 `json:"produces"`
	Cost              costJSON `json:"cost"`
	BuildStamina      int      `json:"build_stamina"`
}

var buildingConfigs map[uint32]BuildingConfig

func init() {
	var entries []buildingConfigJSON
	if err := json.Unmarshal(buildingsConfigJSON, &entries); err != nil {
		panic(fmt.Sprintf("failed to parse config/buildings.json: %v", err))
	}
	buildingConfigs = make(map[uint32]BuildingConfig, len(entries))
	for _, e := range entries {
		buildingConfigs[e.BuildingId] = BuildingConfig{
			display_name:       e.DisplayName,
			max_health:         e.MaxHealth,
			population_support: e.PopulationSupport,
			produces:           e.Produces,
			cost: RisqResourceCost{
				food:  e.Cost.Food,
				wood:  e.Cost.Wood,
				stone: e.Cost.Stone,
			},
			build_stamina: e.BuildStamina,
		}
	}
}
