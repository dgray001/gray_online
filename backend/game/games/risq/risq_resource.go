package risq

import (
	"fmt"
	"os"

	"github.com/gin-gonic/gin"
)

type RisqResource struct {
	internal_id       uint64
	resource_id       uint32
	display_name      string
	zone              *RisqZone
	resources_left    int
	base_gather_speed int
}

func createRisqResource(internal_id uint64, resource_id uint32) *RisqResource {
	resource := RisqResource{
		internal_id: internal_id,
		resource_id: resource_id,
	}
	switch resource_id {
	// food
	case 1: // forage bush
		resource.display_name = "Forage Bushes"
		resource.resources_left = 250
		resource.base_gather_speed = 8
	case 2: // deer
		resource.display_name = "Deer"
		resource.resources_left = 150
		resource.base_gather_speed = 12
	// wood
	case 11: // cedar grove
		resource.display_name = "Cedar Grove"
		resource.resources_left = 650
		resource.base_gather_speed = 9
	case 12: // dead grove
		resource.display_name = "Dead Grove"
		resource.resources_left = 350
		resource.base_gather_speed = 11
	case 13: // maple grove
		resource.display_name = "Maple Grove"
		resource.resources_left = 500
		resource.base_gather_speed = 10
	case 14: // oak grove
		resource.display_name = "Oak Grove"
		resource.resources_left = 650
		resource.base_gather_speed = 9
	case 15: // pine grove
		resource.display_name = "Pine Grove"
		resource.resources_left = 450
		resource.base_gather_speed = 11
	case 16: // walnut grove
		resource.display_name = "Walnut Grove"
		resource.resources_left = 550
		resource.base_gather_speed = 10
	// stone
	case 21: // stonemine
		resource.display_name = "Stone Mine"
		resource.resources_left = 250
		resource.base_gather_speed = 10
	// gold
	case 31: // goldmine
		resource.display_name = "Gold Mine"
		resource.resources_left = 250
		resource.base_gather_speed = 10
	default:
		fmt.Fprintln(os.Stderr, "Invalid resource id: ", resource_id)
	}
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
