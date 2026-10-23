package ai

const (
	unitUIDBase     uint64 = 0
	buildingUIDBase uint64 = 1_000_000_000
	resourceUIDBase uint64 = 2_000_000_000
)

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
