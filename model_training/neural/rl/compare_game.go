package rl

import (
	"fiddlesticks.live/neural/experiment"
	"github.com/dgray001/gray_online/game"
	"github.com/dgray001/gray_online/game/games/fiddlesticks"
	"math/rand"
)

func comparisonGame(candidate benchmarkAgent, scenario experiment.Scenario, audit *PolicyAudit) Episode {
	base := game.CreateBaseGame(1, game.GameType_FIDDLESTICKS, nil, nil, nil)
	f := fiddlesticks.InitializeGame(base)
	a := setupAgentsWithModel(f, candidate.Network, scenario, false, nil, candidate.Model)
	a.RandomCards, a.ZeroBid = candidate.RandomCards, candidate.ZeroBid
	if scenario.Settings.Players >= 4 && scenario.Settings.Players <= 8 {
		a.Audit = audit
		if candidate.Network == nil {
			p := f.GetFiddlesticksPlayer(scenario.Seat)
			p.SetAiModel(&cardAuditor{Model: p.AiModel(), Audit: audit})
		}
	}
	rand.Seed(scenario.ActionSeed + int64(scenario.Seat+1)*1009)
	return executeGame(f, base, a, scenario, nil)
}
