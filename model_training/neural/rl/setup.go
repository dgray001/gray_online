package rl

import (
	"fiddlesticks.live/neural"
	"fiddlesticks.live/neural/experiment"
	"github.com/dgray001/gray_online/game/games/fiddlesticks"
	"math/rand"
)

func setupAgents(f *fiddlesticks.GameFiddlesticks, network *neural.Network, scenario experiment.Scenario, training bool, snapshot *neural.Network) *Agent {
	return setupAgentsWithModel(f, network, scenario, training, snapshot, 4)
}

func setupAgentsWithModel(f *fiddlesticks.GameFiddlesticks, network *neural.Network, scenario experiment.Scenario, training bool, snapshot *neural.Network, focusModel uint8) *Agent {
	var learner *Agent
	for seat := 0; seat < scenario.Settings.Players; seat++ {
		model := scenario.Settings.OpponentModels[seat%len(scenario.Settings.OpponentModels)]
		if seat == scenario.Seat {
			model = focusModel
		}
		f.AddInternalAiPlayer(model, nil)
		agent := &Agent{Random: rand.New(rand.NewSource(scenario.ActionSeed + int64(seat+1)*1009))}
		if seat == scenario.Seat {
			agent.Network, agent.Record, learner = network, training, agent
			if network != nil {
				f.GetFiddlesticksPlayer(seat).SetAiModel(agent)
			}
		} else if scenario.Opponents == "past-self" {
			agent.Network = snapshot
			f.GetFiddlesticksPlayer(seat).SetAiModel(agent)
		} else if model == 0 {
			f.GetFiddlesticksPlayer(seat).SetAiModel(agent)
		}
	}
	return learner
}
