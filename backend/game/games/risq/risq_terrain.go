package risq

import (
	_ "embed"
	"encoding/json"
	"fmt"
)

//go:embed config/terrains.json
var terrainsConfigJSON []byte

const defaultTerrainId uint32 = 1

type TerrainType uint8

const (
	TerrainType_NONE TerrainType = iota
	TerrainType_FLATLANDS
	TerrainType_HILLY
	TerrainType_MOUNTAINOUS
	TerrainType_SWAMP
	TerrainType_SHALLOWS
	TerrainType_WATER
	TerrainType_DEEP_WATER
)

func parseTerrainType(s string) (TerrainType, error) {
	switch s {
	case "flatlands":
		return TerrainType_FLATLANDS, nil
	case "hilly":
		return TerrainType_HILLY, nil
	case "mountainous":
		return TerrainType_MOUNTAINOUS, nil
	case "swamp":
		return TerrainType_SWAMP, nil
	case "shallows":
		return TerrainType_SHALLOWS, nil
	case "water":
		return TerrainType_WATER, nil
	case "deep_water":
		return TerrainType_DEEP_WATER, nil
	default:
		return TerrainType_NONE, fmt.Errorf("unknown terrain_type %q", s)
	}
}

type TerrainConfig struct {
	display_name string
	terrain_type TerrainType
}

type terrainConfigJSON struct {
	TerrainId   uint32 `json:"terrain_id"`
	DisplayName string `json:"display_name"`
	TerrainType string `json:"terrain_type"`
}

var terrainConfigs map[uint32]TerrainConfig

func init() {
	var entries []terrainConfigJSON
	if err := json.Unmarshal(terrainsConfigJSON, &entries); err != nil {
		panic(fmt.Sprintf("failed to parse config/terrains.json: %v", err))
	}
	terrainConfigs = make(map[uint32]TerrainConfig, len(entries))
	for _, e := range entries {
		terrain_type, err := parseTerrainType(e.TerrainType)
		if err != nil {
			panic(fmt.Sprintf("config/terrains.json terrain_id %d: %v", e.TerrainId, err))
		}
		terrainConfigs[e.TerrainId] = TerrainConfig{
			display_name: e.DisplayName,
			terrain_type: terrain_type,
		}
	}
}
