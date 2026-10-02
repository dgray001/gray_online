package risq

import "github.com/dgray001/gray_online/game/games/risq/internal/defs"

type GameMetrics struct {
	Turns  []TurnMetrics
	Fights []FightMetrics
}

type gameMetrics struct {
	turns      []TurnMetrics
	frames     []FightTickMetrics
	actors     map[Attackable]*metricActor
	hits       []metricHit
	parents    []int
	lastSpaces map[*RisqSpace]int
	spaceTurns map[*RisqSpace]uint16
	lastTurns  map[int]uint16
	idleFrames map[uint64]int
}

func (m *gameMetrics) recordTurn(r *GameRisq) {
	t := TurnMetrics{Turn: r.turn_number}
	for id, p := range r.players {
		s := PlayerTurnMetrics{PlayerId: id,
			Stockpile: [4]float64{p.resources.food, p.resources.wood, p.resources.stone, p.resources.gold},
			Income:    p.resources.gathered, Spent: p.resources.spent}
		for _, b := range p.buildings {
			if !b.deleted && b.building_id == villageCenterBuildingId && !b.underConstruction() {
				s.VillageCenters++
			}
		}
		for _, u := range p.units {
			if !u.deleted && u.unitType() == defs.UnitType_ECONOMIC {
				s.Villagers++
			}
		}
		t.Players = append(t.Players, s)
	}
	m.turns = append(m.turns, t)
	m.actors, m.hits = nil, nil
}
