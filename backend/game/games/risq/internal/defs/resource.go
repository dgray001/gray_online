package defs

import (
	"encoding/json"
	"fmt"
	"github.com/gin-gonic/gin"
)

type RisqResourceCategory uint8

const (
	RisqResourceCategory_NONE RisqResourceCategory = iota
	RisqResourceCategory_FOOD
	RisqResourceCategory_WOOD
	RisqResourceCategory_STONE
	RisqResourceCategory_GOLD
	RisqResourceCategory_END
)

func (c RisqResourceCategory) Index() int {
	return int(c) - 1
}

type ResourceConfig struct {
	Display_name       string
	Category           RisqResourceCategory
	Starting_resources float64
	Base_gather_speed  int
	Gather_capacity    int
}

type resourceConfigJSON struct {
	ResourceId        uint32  `json:"resource_id"`
	DisplayName       string  `json:"display_name"`
	Category          string  `json:"category"`
	StartingResources float64 `json:"starting_resources"`
	BaseGatherSpeed   int     `json:"base_gather_speed"`
	GatherCapacity    int     `json:"gather_capacity"`
}

var ResourceConfigs map[uint32]ResourceConfig

func loadResourceConfig(data []byte) {
	var entries []resourceConfigJSON
	if err := json.Unmarshal(data, &entries); err != nil {
		panic(fmt.Sprintf("failed to parse config/resources.json: %v", err))
	}
	ResourceConfigs = make(map[uint32]ResourceConfig, len(entries))
	for _, e := range entries {
		category, err := parseResourceCategory(e.Category)
		if err != nil {
			panic(fmt.Sprintf("config/resources.json resource_id %d: %v", e.ResourceId, err))
		}
		ResourceConfigs[e.ResourceId] = ResourceConfig{
			Display_name:       e.DisplayName,
			Category:           category,
			Starting_resources: e.StartingResources,
			Base_gather_speed:  e.BaseGatherSpeed,
			Gather_capacity:    e.GatherCapacity,
		}
	}
}

func parseResourceCategory(s string) (RisqResourceCategory, error) {
	switch s {
	case "food":
		return RisqResourceCategory_FOOD, nil
	case "wood":
		return RisqResourceCategory_WOOD, nil
	case "stone":
		return RisqResourceCategory_STONE, nil
	case "gold":
		return RisqResourceCategory_GOLD, nil
	default:
		return RisqResourceCategory_NONE, fmt.Errorf("unknown category %q", s)
	}
}

type RisqResourceCost struct {
	Food  float64
	Wood  float64
	Stone float64
	Gold  float64
}

func (c RisqResourceCost) ToFrontend() gin.H {
	return gin.H{
		"food":  c.Food,
		"wood":  c.Wood,
		"stone": c.Stone,
		"gold":  c.Gold,
	}
}

func (c RisqResourceCost) Scale(f float64) RisqResourceCost {
	return RisqResourceCost{
		Food:  c.Food * f,
		Wood:  c.Wood * f,
		Stone: c.Stone * f,
		Gold:  c.Gold * f,
	}
}

func (c RisqResourceCost) Points() uint {
	return uint(c.Food + c.Wood + c.Stone + 2*c.Gold)
}
