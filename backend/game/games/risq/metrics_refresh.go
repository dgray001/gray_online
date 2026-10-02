package risq

import "github.com/dgray001/gray_online/game/games/risq/internal/defs"

func (m *gameMetrics) recordRefresh(r *GameRisq) {
	t := &m.turns[len(m.turns)-1]
	for id, p := range r.players {
		s := &t.Players[id]
		for _, u := range p.units {
			if !u.deleted && u.unitType() == defs.UnitType_ECONOMIC {
				s.VillagerStamina.Granted += u.turn_stamina
				s.VillagerStamina.Wasted += refreshWaste(&u.orderableBase)
			}
		}
		for _, b := range p.buildings {
			if !b.deleted && b.building_id == villageCenterBuildingId && !b.underConstruction() {
				s.VillageCenterStamina.Granted += b.turn_stamina
				s.VillageCenterStamina.Wasted += refreshWaste(&b.orderableBase)
			}
		}
	}
	m.recordFightIdle(r)
	m.idleFrames = nil
}
