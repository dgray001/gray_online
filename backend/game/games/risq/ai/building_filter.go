package ai

type buildingFilter struct {
	ids map[uint32]bool
}

func (f buildingFilter) matches(building_id uint32) bool {
	return len(f.ids) == 0 || f.ids[building_id]
}
