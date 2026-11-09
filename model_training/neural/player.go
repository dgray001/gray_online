package neural

import (
	"math"
	"math/rand"
	"slices"

	"github.com/dgray001/gray_online/game"
	"github.com/dgray001/gray_online/game/games/fiddlesticks"
)

type Example struct {
	Input        []float64
	Legal        []int
	Target       int
	Distribution []float64
}
type Player struct {
	Network  *Network
	Teacher  fiddlesticks.FiddlesticksAiModel
	Examples *[]Example
}

func (m *Player) ApplyUpdate(p *fiddlesticks.FiddlesticksPlayer, f *fiddlesticks.GameFiddlesticks, update *game.UpdateMessage) {
	if m.Teacher != nil {
		m.Teacher.ApplyUpdate(p, f, update)
	}
}

func (m *Player) Bet(p *fiddlesticks.FiddlesticksPlayer, f *fiddlesticks.GameFiddlesticks) float64 {
	state := f.LearningState(p)
	input := FeaturesFor(m.Network, state, true)
	legal := make([]int, int(state.Round)+1)
	for i := range legal {
		legal[i] = i
	}
	if m.Examples != nil {
		bid := m.Teacher.Bet(p, f)
		*m.Examples = append(*m.Examples, Example{input, legal, int(math.Round(bid)), bidDistribution(bid, len(legal))})
		return bid
	}
	probabilities := m.Network.Predict(input, legal)
	if m.Network.BidPolicy == "categorical" {
		value := rand.Float64()
		for _, action := range legal {
			value -= probabilities[action]
			if value < 0 {
				return float64(action)
			}
		}
		return float64(legal[len(legal)-1])
	}
	bid := 0.0
	for _, action := range legal {
		bid += float64(action) * probabilities[action]
	}
	return bid
}

func (m *Player) CardWeights(p *fiddlesticks.FiddlesticksPlayer, f *fiddlesticks.GameFiddlesticks, valid []int) []float64 {
	state := f.LearningState(p)
	input := FeaturesFor(m.Network, state, false)
	legal := make([]int, len(valid))
	for i, index := range valid {
		legal[i] = 27 + cardSlot(state.Hand[index])
	}
	if m.Examples != nil {
		weights := m.Teacher.CardWeights(p, f, valid)
		target := slices.Index(weights, slices.Max(weights))
		*m.Examples = append(*m.Examples, Example{input, legal, legal[target], normalized(weights)})
		return weights
	}
	probabilities := m.Network.Predict(input, legal)
	weights := make([]float64, len(valid))
	for i, action := range legal {
		weights[i] = probabilities[action]
	}
	return weights
}
