package ai

import "slices"

// How many of my units that are not garrisoned stand in each space
func ownUnitsBySpace(view View) map[Coordinate]int {
	counts := map[Coordinate]int{}
	for _, unit := range view.Units() {
		if unit.GarrisonedIn == nil {
			counts[unit.Location.Space]++
		}
	}
	return counts
}

func unidentifiedUnitCount(space SpaceInfo, own map[Coordinate]int) int {
	if space.UnitCount == nil {
		return 0
	}
	return max(0, *space.UnitCount-own[space.Space])
}

func coordinateLess(a Coordinate, b Coordinate) bool {
	if a.X != b.X {
		return a.X < b.X
	}
	return a.Y < b.Y
}

type unitFilter struct {
	unit_ids   map[uint32]bool
	unit_types map[UnitType]bool
}

// With no ids or types set every unit matches
func (f unitFilter) matches(u UnitView) bool {
	return len(f.unit_ids) == 0 && len(f.unit_types) == 0 || f.unit_ids[u.UnitID] || f.unit_types[u.Type]
}

func canBuild(u UnitView, building_id uint32) bool {
	return slices.ContainsFunc(u.Builds, func(p Producible) bool { return p.ID == building_id })
}
