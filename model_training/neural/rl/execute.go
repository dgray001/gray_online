package rl

import (
	"fiddlesticks.live/neural/experiment"
	"github.com/dgray001/gray_online/game"
	"github.com/dgray001/gray_online/game/games/fiddlesticks"
)

func executeGame(f *fiddlesticks.GameFiddlesticks, base *game.GameBase, agent *Agent, scenario experiment.Scenario, learner *Learner) Episode {
	episode := Episode{Scenario: scenario}
	f.SetRoundObserver(func(outcomes []fiddlesticks.RoundOutcome) {
		episode.Outcomes = append(episode.Outcomes, outcomes[scenario.Seat])
		for _, outcome := range outcomes {
			if outcome.Seat != scenario.Seat {
				episode.OpponentRounds++
				if outcome.Bid == outcome.Tricks {
					episode.OpponentExact++
				}
			}
		}
		if learner != nil {
			learner.complete(agent, outcomes[scenario.Seat], scenario.Settings.Players)
		}
	})
	s := scenario.Settings
	f.SetSettings(1, s.MaxRound, s.RoundPoints, s.TrickPoints, 1)
	f.SetDealSeed(s.DealSeed)
	f.SetActionSeed(scenario.ActionSeed)
	base.StartGame()
	f.StartAiGame(false)
	for !f.ExecuteAiTurn(false) {
	}
	_, episode.Scores = f.GetGameResults()
	episode.Decisions = agent.Decisions
	return episode
}
