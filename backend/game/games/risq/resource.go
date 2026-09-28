package risq

import (
	"fmt"
	"os"

	"github.com/dgray001/gray_online/game/games/risq/internal/defs"
	"github.com/gin-gonic/gin"
)

type RisqResource struct {
	internal_id       uint64
	resource_id       uint32
	display_name      string
	zone              *RisqZone
	resources_left    float64
	base_gather_speed int
	gather_capacity   int
	resource_category defs.RisqResourceCategory
}

func (r *RisqResource) category() defs.RisqResourceCategory {
	return r.resource_category
}

func createRisqResource(internal_id uint64, resource_id uint32) *RisqResource {
	resource := RisqResource{
		internal_id: internal_id,
		resource_id: resource_id,
	}
	config, ok := defs.ResourceConfigs[resource_id]
	if !ok {
		fmt.Fprintln(os.Stderr, "Invalid resource id: ", resource_id)
		return &resource
	}
	resource.display_name = config.Display_name
	resource.resources_left = config.Starting_resources
	resource.base_gather_speed = config.Base_gather_speed
	resource.gather_capacity = config.Gather_capacity
	resource.resource_category = config.Category
	return &resource
}

func (r *RisqResource) toFrontend() gin.H {
	resource := gin.H{
		"internal_id":       r.internal_id,
		"resource_id":       r.resource_id,
		"display_name":      r.display_name,
		"resources_left":    r.resources_left,
		"base_gather_speed": r.base_gather_speed,
		"gather_capacity":   r.gather_capacity,
	}
	if r.zone != nil {
		resource["zone_coordinate"] = r.zone.coordinate.ToFrontend()
		if r.zone.space != nil {
			resource["space_coordinate"] = r.zone.space.coordinate.ToFrontend()
		}
	}
	return resource
}
