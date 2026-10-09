package risq

import (
	"cmp"
	"encoding/json"
	"github.com/dgray001/gray_online/game/games/risq/internal/defs"
	"github.com/dgray001/gray_online/util"
	"io"
	"slices"
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

// Debug log only: each player's finished turn, the metrics record and the turn report, so one run answers per-turn questions
func (m *gameMetrics) logTurn(r *GameRisq, t *TurnMetrics) {
	if util.DebugLog.Writer() == io.Discard {
		return
	}
	for id, p := range r.players {
		metrics, _ := json.Marshal(t.Players[id])
		report, _ := json.Marshal(p.report.toFrontend())
		util.DebugLog.Printf("turn report turn=%d player=%d metrics=%s report=%s", t.Turn, id, metrics, report)
	}
}

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
			if m.extras["production"] && isProductionBuilding(b) {
				production := s.production(b.building_id)
				production.Stamina.Granted += b.turn_stamina
				production.Stamina.Wasted += refreshWaste(&b.orderableBase)
			}
		}
	}
	m.logTurn(r, t)
	m.recordFightIdle(r)
	m.idleFrames = nil
}
