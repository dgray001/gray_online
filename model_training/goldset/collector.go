package goldset

import (
	"github.com/dgray001/gray_online/game"
	"github.com/dgray001/gray_online/game/games/fiddlesticks"
)

type collector struct {
	Model   *fiddlesticks.FiddlesticksAiModelTheory4
	Capture func(*fiddlesticks.FiddlesticksPlayer, *fiddlesticks.GameFiddlesticks) *Item
	Accept  func(Item)
	Pending *Item
}

func (m *collector) Bet(p *fiddlesticks.FiddlesticksPlayer, f *fiddlesticks.GameFiddlesticks) float64 {
	bid := m.Model.Bet(p, f)
	m.Pending = m.Capture(p, f)
	return bid
}
func (m *collector) CardWeights(p *fiddlesticks.FiddlesticksPlayer, f *fiddlesticks.GameFiddlesticks, valid []int) []float64 {
	weights := m.Model.CardWeights(p, f, valid)
	if len(valid) > 1 {
		m.Pending = m.Capture(p, f)
	}
	return weights
}

func (m *collector) ApplyUpdate(p *fiddlesticks.FiddlesticksPlayer, f *fiddlesticks.GameFiddlesticks, update *game.UpdateMessage) {
	if m.Pending != nil && (update.Kind == "bet" || update.Kind == "play-card") && update.Content["player_id"].(int) == m.Pending.State.Seat {
		item := *m.Pending
		if update.Kind == "bet" {
			item.Model4Action = int(update.Content["amount"].(uint8))
		} else {
			index := update.Content["index"].(int)
			item.Model4Action = item.State.Hand[index].Action
		}
		m.Accept(item)
		m.Pending = nil
	}
	m.Model.ApplyUpdate(p, f, update)
}
