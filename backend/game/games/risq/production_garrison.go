package risq

func (r *GameRisq) computeProductionGarrisons() map[*RisqBuilding]bool {
	reserved := make(map[*RisqBuilding]int)
	for unit := range r.garrison_allotments {
		reserved[unit.intent.detail.(*GarrisonIntent).target]++
	}
	garrisons := make(map[*RisqBuilding]bool)
	for building := range r.unit_creation_ids {
		point := building.gather_point
		if point == nil || point.location_kind != RisqGatherPointLocationKind_ZONE || point.object_type != RisqGatherObjectType_BUILDING || point.object_id != building.internal_id || point.location_id != uint64(building.zone.coordinate_key) {
			continue
		}
		if len(building.garrisoned_units)+reserved[building] < int(building.garrison_capacity) {
			garrisons[building] = true
		}
	}
	return garrisons
}
