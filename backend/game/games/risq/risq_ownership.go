package risq

const spaceGoldIncome = 2.0

// A space is owned by whoever is the sole building owner there; with no buildings at all,
// ownership instead falls to whoever is the sole owner of military units there (economic units don't conquer).
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
		if unit == nil || unit.deleted || isEconomicUnit(unit.unit_id) {
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

// A player may build in a space they hold, or an unclaimed space bordering one they hold
func (s *RisqSpace) buildableBy(player_id int) bool {
	owner := s.computeOwnership()
	if owner == player_id {
		return true
	}
	if owner != -1 {
		return false
	}
	for _, adj := range s.adjacent_spaces {
		if adj.computeOwnership() == player_id {
			return true
		}
	}
	return false
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
