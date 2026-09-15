package risq

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"os"

	"github.com/gin-gonic/gin"
)

type RisqResource struct {
	internal_id       uint64
	resource_id       uint32
	display_name      string
	zone              *RisqZone
	resources_left    float64
	base_gather_speed int
	resource_category RisqResourceCategory
}

type RisqResourceCategory uint8

const (
	RisqResourceCategory_NONE RisqResourceCategory = iota
	RisqResourceCategory_FOOD
	RisqResourceCategory_WOOD
	RisqResourceCategory_STONE
	RisqResourceCategory_GOLD
	RisqResourceCategory_END
)

func (c RisqResourceCategory) index() int {
	return int(c) - 1
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

func (r *RisqResource) category() RisqResourceCategory {
	return r.resource_category
}

//go:embed config/resources.json
var resourcesConfigJSON []byte

type ResourceConfig struct {
	display_name       string
	category           RisqResourceCategory
	starting_resources float64
	base_gather_speed  int
}

type resourceConfigJSON struct {
	ResourceId        uint32  `json:"resource_id"`
	DisplayName       string  `json:"display_name"`
	Category          string  `json:"category"`
	StartingResources float64 `json:"starting_resources"`
	BaseGatherSpeed   int     `json:"base_gather_speed"`
}

var resourceConfigs map[uint32]ResourceConfig

func init() {
	var entries []resourceConfigJSON
	if err := json.Unmarshal(resourcesConfigJSON, &entries); err != nil {
		panic(fmt.Sprintf("failed to parse config/resources.json: %v", err))
	}
	resourceConfigs = make(map[uint32]ResourceConfig, len(entries))
	for _, e := range entries {
		category, err := parseResourceCategory(e.Category)
		if err != nil {
			panic(fmt.Sprintf("config/resources.json resource_id %d: %v", e.ResourceId, err))
		}
		resourceConfigs[e.ResourceId] = ResourceConfig{
			display_name:       e.DisplayName,
			category:           category,
			starting_resources: e.StartingResources,
			base_gather_speed:  e.BaseGatherSpeed,
		}
	}
}

func createRisqResource(internal_id uint64, resource_id uint32) *RisqResource {
	resource := RisqResource{
		internal_id: internal_id,
		resource_id: resource_id,
	}
	config, ok := resourceConfigs[resource_id]
	if !ok {
		fmt.Fprintln(os.Stderr, "Invalid resource id: ", resource_id)
		return &resource
	}
	resource.display_name = config.display_name
	resource.resources_left = config.starting_resources
	resource.base_gather_speed = config.base_gather_speed
	resource.resource_category = config.category
	return &resource
}

func (r *RisqResource) toFrontend() gin.H {
	resource := gin.H{
		"internal_id":       r.internal_id,
		"resource_id":       r.resource_id,
		"display_name":      r.display_name,
		"resources_left":    r.resources_left,
		"base_gather_speed": r.base_gather_speed,
	}
	if r.zone != nil {
		resource["zone_coordinate"] = r.zone.coordinate.ToFrontend()
		if r.zone.space != nil {
			resource["space_coordinate"] = r.zone.space.coordinate.ToFrontend()
		}
	}
	return resource
}
