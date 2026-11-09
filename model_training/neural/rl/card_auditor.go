package rl

import (
	"github.com/dgray001/gray_online/game"
	"github.com/dgray001/gray_online/game/games/fiddlesticks"
)

type cardAuditor struct {
	Model fiddlesticks.FiddlesticksAiModel
	Audit *PolicyAudit
}

func (m *cardAuditor) ApplyUpdate(p *fiddlesticks.FiddlesticksPlayer, f *fiddlesticks.GameFiddlesticks, update *game.UpdateMessage) {
	m.Model.ApplyUpdate(p, f, update)
}
func (m *cardAuditor) Bet(p *fiddlesticks.FiddlesticksPlayer, f *fiddlesticks.GameFiddlesticks) float64 {
	return m.Model.Bet(p, f)
}

func (m *cardAuditor) CardWeights(p *fiddlesticks.FiddlesticksPlayer, f *fiddlesticks.GameFiddlesticks, valid []int) []float64 {
	weights := m.Model.CardWeights(p, f, valid)
	if len(valid) > 1 {
		auditCardWeights(m.Audit, weights)
	}
	return weights
}
