package fakeboard

import (
	"fmt"

	"github.com/dgray001/gray_online/game/games/risq/internal/mapgen"
)

// The engine silently drops a resource placed on an occupied zone, so the fake records it for tests to assert never happens
func (b *Board) PlaceResource(z mapgen.Zone, resource_id uint32) {
	zone := z.(*zone)
	if zone.Occupied() {
		b.Violations = append(b.Violations, fmt.Sprintf("resource %d placed on occupied zone %d", resource_id, zone.key))
		return
	}
	zone.resource = resource_id
}

func (b *Board) PlaceBuilding(z mapgen.Zone, building_id uint32, player_index int, resources_left ...float64) bool {
	zone := z.(*zone)
	if zone.Occupied() {
		return false
	}
	zone.building = &Building{ID: building_id, Player: player_index}
	return true
}

func (b *Board) PlaceUnit(z mapgen.Zone, unit_id uint32, player_index int) {
	zone := z.(*zone)
	zone.units = append(zone.units, Unit{ID: unit_id, Player: player_index})
}
