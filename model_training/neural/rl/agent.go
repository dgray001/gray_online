package rl

import (
	"fiddlesticks.live/neural"
	"github.com/dgray001/gray_online/game"
	"github.com/dgray001/gray_online/game/games/fiddlesticks"
	"math/rand"
)

type Agent struct {
	Network     *neural.Network
	Random      *rand.Rand
	Record      bool
	Steps       []neural.PolicyStep
	Decisions   int
	RandomCards bool
	ZeroBid     bool
	Audit       *PolicyAudit
}

func (a *Agent) ApplyUpdate(p *fiddlesticks.FiddlesticksPlayer, f *fiddlesticks.GameFiddlesticks, update *game.UpdateMessage) {
}

func (a *Agent) Bet(p *fiddlesticks.FiddlesticksPlayer, f *fiddlesticks.GameFiddlesticks) float64 {
	state := f.LearningState(p)
	legal := make([]int, int(state.Round)+1)
	for i := range legal {
		legal[i] = i
	}
	return float64(a.choose(neural.FeaturesFor(a.Network, state, true), legal))
}
