package risq

const spaceGoldIncome = 2.0

func (s *RisqSpace) computeOwnership() int {
	owner := -1
	for _, building := range s.buildings {
		if building == nil || building.deleted {
			continue
		}
		if owner == -1 {
			owner = building.player_id
		} else if owner != building.player_id {
			return -1
		}
	}
	return owner
}

func (r *GameRisq) recalculateOwnership() {
	for _, row := range r.spaces {
		for _, space := range row {
			owner := space.computeOwnership()
			space.ownership = owner
			if owner >= 0 && owner < len(r.players) {
				r.players[owner].resources.addGathered(RisqResourceCategory_GOLD, spaceGoldIncome)
			}
		}
	}
}
