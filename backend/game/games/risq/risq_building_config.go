package risq

import (
	_ "embed"
	"encoding/json"
	"fmt"
)

//go:embed config/buildings.json
var buildingsConfigJSON []byte

type BuildingConfig struct {
	display_name       string
	max_health         int
	population_support uint16
	produces           []uint32
}

func (c BuildingConfig) canProduce(unit_id uint32) bool {
	for _, id := range c.produces {
		if id == unit_id {
			return true
		}
	}
	return false
}

type buildingConfigJSON struct {
	BuildingId        uint32   `json:"building_id"`
	DisplayName       string   `json:"display_name"`
	MaxHealth         int      `json:"max_health"`
	PopulationSupport uint16   `json:"population_support"`
	Produces          []uint32 `json:"produces"`
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
		}
	}
}
