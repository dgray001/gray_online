package aibridge

import "maps"

type idleSnapshot struct {
	units     map[uint64]bool
	buildings map[uint64]bool
}

// IdleSnapshot returns read-only maps that retain their values after later orders.
func (v *aiView) IdleSnapshot() (map[uint64]bool, map[uint64]bool) {
	if v.idle_snapshot.units == nil {
		v.idle_snapshot.units = make(map[uint64]bool)
		v.idle_snapshot.buildings = make(map[uint64]bool)
		for _, unit := range v.IdleUnits() {
			v.idle_snapshot.units[unit.InternalID] = true
		}
		for _, building := range v.IdleBuildings() {
			v.idle_snapshot.buildings[building.InternalID] = true
		}
	}
	return v.idle_snapshot.units, v.idle_snapshot.buildings
}

func withIdleMember(index map[uint64]bool, id uint64, idle bool) map[uint64]bool {
	if index[id] == idle {
		return index
	}
	updated := maps.Clone(index)
	if idle {
		updated[id] = true
	} else {
		delete(updated, id)
	}
	return updated
}

func (v *aiView) refreshIdle(subject uint64) {
	if v.idle_snapshot.units == nil {
		return
	}
	if subject < 1_000_000_000 {
		idle := v.currentOrder(v.queues[subject]) == nil
		v.idle_snapshot.units = withIdleMember(v.idle_snapshot.units, subject, idle)
		return
	}
	id := subject - 1_000_000_000
	idle := !v.claimed_buildings[id] && v.ownBuildingView(v.buildings[id]).Idle
	v.idle_snapshot.buildings = withIdleMember(v.idle_snapshot.buildings, id, idle)
}
