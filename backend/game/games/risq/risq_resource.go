package risq

import (
	"fmt"
	"os"

	"github.com/gin-gonic/gin"
)

type RisqResource struct {
	internal_id    uint64
	resource_id    uint32
	zone           *RisqZone
	resources_left int
}

func createRisqResource(internal_id uint64, resource_id uint32) *RisqResource {
	resource := RisqResource{
		internal_id: internal_id,
		resource_id: resource_id,
	}
	switch resource_id {
	// food
	case 1: // forage bush
		resource.resources_left = 200
	case 2: // deer
		resource.resources_left = 100
	// wood
	case 11: // cedar grove
		resource.resources_left = 650
	case 12: // dead grove
		resource.resources_left = 350
	case 13: // maple grove
		resource.resources_left = 500
	case 14: // oak grove
		resource.resources_left = 650
	case 15: // pine grove
		resource.resources_left = 450
	case 16: // walnut grove
		resource.resources_left = 550
	// stone
	case 21: // stonemine
		resource.resources_left = 150
	// gold
	case 31: // goldmine
		resource.resources_left = 150
	default:
		fmt.Fprintln(os.Stderr, "Invalid resource id: ", resource_id)
	}
	return &resource
}

func (r *RisqResource) toFrontend() gin.H {
	resource := gin.H{
		"internal_id":    r.internal_id,
		"resource_id":    r.resource_id,
		"resources_left": r.resources_left,
	}
	if r.zone != nil {
		resource["zone_coordinate"] = r.zone.coordinate.ToFrontend()
		if r.zone.space != nil {
			resource["space_coordinate"] = r.zone.space.coordinate.ToFrontend()
		}
	}
	return resource
}
