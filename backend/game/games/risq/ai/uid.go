package ai

import "github.com/dgray001/gray_online/util"

const (
	unitUIDBase       uint64 = 0
	buildingUIDBase   uint64 = 1_000_000_000
	resourceUIDBase   uint64 = 2_000_000_000
	foundationUIDBase uint64 = 3_000_000_000
)

// A foundation has no internal id, so its uid pairs its zone key with the owning player
func FoundationUID(location ZoneRef, player_id int) uint64 {
	space_key := util.Pair(location.Space.X, location.Space.Y)
	zone_key := util.Pair(int(space_key), int(util.Pair(location.Zone.X, location.Zone.Y)))
	return foundationUIDBase + uint64(util.Pair(int(zone_key), player_id))
}

func unitUID(internal_id uint64) uint64 {
	return unitUIDBase + internal_id
}

func buildingUID(internal_id uint64) uint64 {
	return buildingUIDBase + internal_id
}

func resourceUID(internal_id uint64) uint64 {
	return resourceUIDBase + internal_id
}

func (r ResourceView) uid() uint64 {
	if r.IsBuilding {
		return buildingUID(r.InternalID)
	}
	return resourceUID(r.InternalID)
}
