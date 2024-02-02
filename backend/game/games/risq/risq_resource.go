package risq

import "github.com/gin-gonic/gin"

type RisqResource struct {
	internal_id    uint64
	resource_id    uint32
	zone           *RisqZone
	resources_left int
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
