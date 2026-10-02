package risq

import (
	"github.com/dgray001/gray_online/game/games/risq/internal/defs"
	"github.com/dgray001/gray_online/util"
)

type metricActor struct {
	actor                          Attackable
	base                           *orderableBase
	unit                           *RisqUnit
	space                          *RisqSpace
	attackTarget                   *RisqSpace
	health                         float64
	stamina                        int
	moving, military               bool
	attackStamina, overkillStamina float64
}

type metricHit struct {
	attacker, target *metricActor
	damage           float64
	stamina          int
}

func (m *gameMetrics) beginTick(r *GameRisq, orderables []Orderable) {
	m.actors = make(map[Attackable]*metricActor, len(orderables))
	m.hits = m.hits[:0]
	for _, o := range orderables {
		s := &metricActor{}
		switch a := o.(type) {
		case *RisqUnit:
			s.actor, s.base, s.unit = a, &a.orderableBase, a
		case *RisqBuilding:
			s.actor, s.base = a, &a.orderableBase
		default:
			continue
		}
		if s.base.deleted || s.base.cs.health <= 0 {
			continue
		}
		zone := s.base.zone
		if zone == nil && s.unit != nil && s.unit.garrisoned_in != nil {
			zone = s.unit.garrisoned_in.zone
		}
		if zone == nil {
			continue
		}
		s.space, s.health, s.stamina = zone.space, s.base.cs.health, s.base.current_stamina
		_, s.moving = s.base.intent.detail.(*MoveIntent)
		s.military = s.unit != nil && s.unit.unitType() != defs.UnitType_ECONOMIC && s.unit.unitType() != defs.UnitType_NONE
		m.actors[s.actor] = s
	}
	for _, a := range m.actors {
		if a.military {
			a.attackTarget = metricAttackTarget(a, r)
		}
	}
}

func (m *gameMetrics) recordAttack(attacker Attackable, target Attackable, damage float64, stamina int) {
	key := attacker
	if g, ok := attacker.(garrisonAttacker); ok {
		key = g.RisqUnit
	}
	a, t := m.actors[key], m.actors[target]
	if a == nil || t == nil {
		return
	}
	if _, garrisoned := attacker.(garrisonAttacker); garrisoned {
		a.actor = attacker
	}
	a.attackStamina += float64(stamina)
	m.hits = append(m.hits, metricHit{attacker: a, target: t, damage: damage, stamina: stamina})
}

func (m *gameMetrics) resolveOverkill() {
	damage := make(map[*metricActor]float64)
	for _, h := range m.hits {
		damage[h.target] += h.damage
	}
	for _, h := range m.hits {
		total := damage[h.target]
		if total <= 0 {
			continue
		}
		healing := max(0, h.target.base.cs.pending_health_delta+total)
		if b, ok := h.target.actor.(*RisqBuilding); ok && b.stamina_remaining != b.health_synced_stamina {
			before := constructionHealthRatio(b.health_synced_stamina, b.construction_stamina_total)
			after := constructionHealthRatio(b.stamina_remaining, b.construction_stamina_total)
			healing += util.RoundTo(float64(b.cs.max_health)*(after-before), gatherRoundingPlaces)
		}
		waste := max(0, total-h.target.health-healing) / total
		h.attacker.overkillStamina += float64(h.stamina) * waste
	}
}
