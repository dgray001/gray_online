package risq

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"math/rand"
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

type TerrainMoveCost struct {
	intra_cost uint
	inter_cost uint
	impassable bool
}

type terrainEntryJSON struct {
	TerrainId   uint32 `json:"terrain_id"`
	DisplayName string `json:"display_name"`
}

type terrainTypeJSON struct {
	TerrainType string             `json:"terrain_type"`
	IntraCost   uint               `json:"intra_cost"`
	InterCost   uint               `json:"inter_cost"`
	Impassable  bool               `json:"impassable"`
	Terrains    []terrainEntryJSON `json:"terrains"`
}

var terrainConfigs map[uint32]TerrainConfig
var terrainMoveCosts map[TerrainType]TerrainMoveCost
var terrainIdsByType map[TerrainType][]uint32

var lowestIntraMoveCost uint = 1
var lowestInterMoveCost uint = 1

func init() {
	var types []terrainTypeJSON
	if err := json.Unmarshal(terrainsConfigJSON, &types); err != nil {
		panic(fmt.Sprintf("failed to parse config/terrains.json: %v", err))
	}
	terrainConfigs = make(map[uint32]TerrainConfig)
	terrainMoveCosts = make(map[TerrainType]TerrainMoveCost, len(types))
	terrainIdsByType = make(map[TerrainType][]uint32, len(types))
	for _, t := range types {
		terrain_type, err := parseTerrainType(t.TerrainType)
		if err != nil {
			panic(fmt.Sprintf("config/terrains.json terrain_type %q: %v", t.TerrainType, err))
		}
		if !t.Impassable {
			if t.IntraCost < 1 || t.InterCost < 1 {
				panic(fmt.Sprintf("config/terrains.json terrain_type %q: move costs must be at least 1", t.TerrainType))
			}
			lowestIntraMoveCost = min(lowestIntraMoveCost, t.IntraCost)
			lowestInterMoveCost = min(lowestInterMoveCost, t.InterCost)
		}
		terrainMoveCosts[terrain_type] = TerrainMoveCost{intra_cost: t.IntraCost, inter_cost: t.InterCost, impassable: t.Impassable}
		for _, e := range t.Terrains {
			terrainConfigs[e.TerrainId] = TerrainConfig{
				display_name: e.DisplayName,
				terrain_type: terrain_type,
			}
			terrainIdsByType[terrain_type] = append(terrainIdsByType[terrain_type], e.TerrainId)
		}
	}
}

func randomTerrainId(terrain_type TerrainType, rng *rand.Rand) uint32 {
	ids := terrainIdsByType[terrain_type]
	if len(ids) == 0 {
		return defaultTerrainId
	}
	return ids[rng.Intn(len(ids))]
}

func (t TerrainType) moveCost() TerrainMoveCost {
	return terrainMoveCosts[t]
}

func (s *RisqSpace) terrainType() TerrainType {
	return terrainConfigs[s.terrain_id].terrain_type
}

func (s *RisqSpace) impassable() bool {
	return s.terrainType().moveCost().impassable
}
