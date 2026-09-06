package risq

const spaceGoldIncome = 2.0

// A space is owned by whoever is the sole building owner there; with no buildings at all,
// ownership instead falls to whoever is the sole unit owner there.
func (s *RisqSpace) computeOwnership() int {
	building_owner := -1
	for _, building := range s.buildings {
		if building == nil || building.deleted {
			continue
		}
		if building_owner == -1 {
			building_owner = building.player_id
		} else if building_owner != building.player_id {
			return -1
		}
	}
	if building_owner != -1 {
		return building_owner
	}
	unit_owner := -1
	for _, unit := range s.units {
		if unit == nil || unit.deleted {
			continue
		}
		if unit_owner == -1 {
			unit_owner = unit.player_id
		} else if unit_owner != unit.player_id {
			return -1
		}
	}
	return unit_owner
}

func (r *GameRisq) recalculateOwnership() {
	for _, row := range r.spaces {
		for _, space := range row {
			owner := space.computeOwnership()
			space.ownership = owner
			if owner >= 0 && owner < len(r.players) {
				r.players[owner].resources.addGathered(RisqResourceCategory_GOLD, spaceGoldIncome)
				r.players[owner].report.recordGoldFromLand(spaceGoldIncome)
			}
		}
	}
}
