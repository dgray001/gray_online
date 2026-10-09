package risq

import (
	"slices"
)

func (m *gameMetrics) recordTick(r *GameRisq, tick uint16) {
	m.resolveOverkill()
	for _, group := range combatGroups(m.combatSpaces(r.turn_number)) {
		spaces, adjacent := make(map[*RisqSpace]bool), make(map[*RisqSpace]bool)
		frame := FightTickMetrics{Turn: r.turn_number, Tick: tick, Players: make([]FightPlayerMetrics, len(r.players))}
		for _, space := range group {
			spaces[space] = true
			frame.Spaces = append(frame.Spaces, [2]int{space.coordinate.X, space.coordinate.Y})
			for _, neighbor := range space.adjacent_spaces {
				adjacent[neighbor] = true
			}
		}
		frame.FightId = m.fightId(spaces, r.turn_number)
		for id := range frame.Players {
			frame.Players[id] = FightPlayerMetrics{PlayerId: id, Military: make(map[uint32]int), Adjacent: make(map[uint32]int)}
		}
		m.fillFight(&frame, spaces, adjacent)
		m.frames = append(m.frames, frame)
	}
}

func (m *gameMetrics) fillFight(frame *FightTickMetrics, spaces, adjacent map[*RisqSpace]bool) {
	if m.idleFrames == nil {
		m.idleFrames = make(map[uint64]int)
	}
	for _, a := range m.actors {
		p := &frame.Players[a.base.player_id]
		if a.military && (spaces[a.space] || adjacent[a.space]) &&
			(spaces[a.attackTarget] || a.attackStamina > 0) {
			p.Engaging = append(p.Engaging, a.unit.internal_id)
		}
		if b, ok := a.actor.(*RisqBuilding); ok {
			p.Buildings = append(p.Buildings, BuildingLocationMetrics{InternalId: b.internal_id,
				BuildingId: b.building_id, Space: [2]int{a.space.coordinate.X, a.space.coordinate.Y}})
		}
		if !spaces[a.space] {
			if a.military && adjacent[a.space] {
				p.Adjacent[a.unit.unit_id]++
			}
			continue
		}
		p.Combatants++
		p.Health += a.health
		if !a.military {
			continue
		}
		id := a.unit.internal_id
		p.Military[a.unit.unit_id]++
		p.Units = append(p.Units, id)
		m.idleFrames[id] = len(m.frames)
		if a.attackStamina > 0 {
			p.Attacking = append(p.Attacking, id)
		} else {
			p.NotAttacking = append(p.NotAttacking, id)
		}
		if a.moving {
			p.Moving = append(p.Moving, id)
		}
		if a.stamina == 0 {
			p.Exhausted = append(p.Exhausted, id)
		}
		if a.unit.garrisoned_in != nil {
			p.Garrisoned = append(p.Garrisoned, id)
		}
		spent := float64(max(0, a.stamina-a.base.current_stamina))
		p.AttackStamina += a.attackStamina
		p.OverkillStamina += a.overkillStamina
		p.MoveStamina += float64(a.sunkStamina)
		if a.moving {
			p.MoveStamina += spent
		} else {
			p.OtherStamina += max(0, spent-a.attackStamina-float64(a.sunkStamina))
		}
	}
	for id := range frame.Players {
		p := &frame.Players[id]
		slices.SortFunc(p.Buildings, func(a, b BuildingLocationMetrics) int {
			if a.InternalId < b.InternalId {
				return -1
			}
			if a.InternalId > b.InternalId {
				return 1
			}
			return 0
		})
		for _, ids := range [][]uint64{p.Engaging, p.Units, p.Attacking, p.NotAttacking, p.Moving, p.Exhausted, p.Garrisoned} {
			slices.Sort(ids)
		}
	}
}

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
