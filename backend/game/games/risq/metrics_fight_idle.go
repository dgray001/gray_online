package risq

import "slices"

func (m *gameMetrics) recordFightIdle(r *GameRisq) {
	for id, index := range m.idleFrames {
		u := r.units[id]
		if u == nil || u.deleted {
			continue
		}
		zone := u.zone
		if zone == nil && u.garrisoned_in != nil {
			zone = u.garrisoned_in.zone
		}
		if zone == nil {
			continue
		}
		frame := &m.frames[index]
		if slices.Contains(frame.Spaces, [2]int{zone.space.coordinate.X, zone.space.coordinate.Y}) {
			frame.Players[u.player_id].IdleStamina += refreshWaste(&u.orderableBase)
		}
	}
}
