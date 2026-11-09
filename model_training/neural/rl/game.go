package rl

import (
	"fiddlesticks.live/neural"
	"fiddlesticks.live/neural/experiment"
	"github.com/dgray001/gray_online/game"
	"github.com/dgray001/gray_online/game/games/fiddlesticks"
)

type Episode struct {
	Scenario       experiment.Scenario
	Scores         []uint32
	Outcomes       []fiddlesticks.RoundOutcome
	OpponentExact  int
	OpponentRounds int
	Decisions      int
}

func play(network *neural.Network, scenario experiment.Scenario, learner *Learner, snapshot *neural.Network) Episode {
	base := game.CreateBaseGame(1, game.GameType_FIDDLESTICKS, nil, nil, nil)
	f := fiddlesticks.InitializeGame(base)
	agent := setupAgents(f, network, scenario, learner != nil, snapshot)
	return executeGame(f, base, agent, scenario, learner)
}
