package risq

import "github.com/dgray001/gray_online/game/games/risq/internal/defs"

// Whether any building exists, or any unit that exists or can be hired is able to construct one
func (r *GameRisq) computeHasBuildings() bool {
	if len(r.buildings) > 0 {
		return true
	}
	builder := func(unit_id uint32) bool {
		return len(r.config.Units[unit_id].Builds) > 0
	}
	for _, unit := range r.units {
		if builder(unit.unit_id) {
			return true
		}
	}
	for _, tech := range r.config.Techs {
		for _, unit_id := range tech.Unlocks_mercenary_ids {
			if builder(unit_id) {
				return true
			}
		}
	}
	return false
}

// Whether any economic unit exists, can be produced by a placed building, or can be hired through a tech
func (r *GameRisq) computeHasEconomy() bool {
	economic := func(unit_id uint32) bool {
		return r.config.Units[unit_id].Unit_type == defs.UnitType_ECONOMIC
	}
	for _, unit := range r.units {
		if economic(unit.unit_id) {
			return true
		}
	}
	for _, building := range r.buildings {
		for _, producible := range r.config.Buildings[building.building_id].Produces {
			if producible.Kind == defs.ProducibleKind_UNIT && economic(producible.Id) {
				return true
			}
		}
	}
	for _, tech := range r.config.Techs {
		for _, unit_id := range tech.Unlocks_mercenary_ids {
			if economic(unit_id) {
				return true
			}
		}
	}
	return false
}
