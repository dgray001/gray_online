package rl

import "fiddlesticks.live/neural"

func trainEpoch(network *neural.Network, trial Trial, learner *Learner, epoch int, options Options) Evaluation {
	scenarios := trainingSchedule(options.GamesPerEpoch, 100000+int64(epoch), options.CoreShare)
	snapshot := network.Clone()
	episodes := make([]Episode, len(scenarios))
	for i, scenario := range scenarios {
		scenario.ActionSeed += trial.Seed * 1000000000000
		if i%5 == 4 {
			scenario.Opponents = "past-self"
		}
		episodes[i] = play(network, scenario, learner, snapshot)
	}
	return summarize(trial.ID, episodes)
}
