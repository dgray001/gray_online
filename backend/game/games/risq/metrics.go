package risq

import (
	"cmp"
	"slices"

	"github.com/dgray001/gray_online/game/games/risq/internal/defs"
)

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
	extras     map[string]bool
}

func isProductionBuilding(b *RisqBuilding) bool {
	return !b.deleted && b.building_id != villageCenterBuildingId && !b.underConstruction() && len(defs.BuildingConfigs[b.building_id].Produces) > 0
}

func (s *PlayerTurnMetrics) production(building_id uint32) *ProductionMetrics {
	for i := range s.Production {
		if s.Production[i].BuildingId == building_id {
			return &s.Production[i]
		}
	}
	s.Production = append(s.Production, ProductionMetrics{BuildingId: building_id})
	return &s.Production[len(s.Production)-1]
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
		if m.extras["production"] {
			for _, b := range p.buildings {
				if isProductionBuilding(b) {
					s.production(b.building_id).Count++
				}
			}
			slices.SortFunc(s.Production, func(a ProductionMetrics, b ProductionMetrics) int { return cmp.Compare(a.BuildingId, b.BuildingId) })
		}
		t.Players = append(t.Players, s)
	}
	m.turns = append(m.turns, t)
	m.actors, m.hits = nil, nil
}
