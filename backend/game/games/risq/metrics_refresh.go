package risq

import (
	"encoding/json"
	"io"

	"github.com/dgray001/gray_online/game/games/risq/internal/defs"
	"github.com/dgray001/gray_online/util"
)

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
