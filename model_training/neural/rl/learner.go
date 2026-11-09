package rl

import "github.com/dgray001/gray_online/game/games/fiddlesticks"

type Context struct {
	Players int
	Round   uint8
}
type Baseline struct {
	Count     int
	Successes float64
}
type Learner struct {
	Rate      float64
	Entropy   float64
	Baselines map[Context]Baseline
}

func (l *Learner) complete(agent *Agent, outcome fiddlesticks.RoundOutcome, players int) {
	key := Context{players, outcome.Round}
	baseline := l.Baselines[key]
	reward := 0.0
	if outcome.Bid == outcome.Tricks {
		reward = 1
	}
	advantage := reward - (baseline.Successes+1)/float64(baseline.Count+2)
	agent.Network.Reinforce(agent.Steps, advantage, l.Rate, l.Entropy)
	baseline.Count++
	baseline.Successes += reward
	l.Baselines[key] = baseline
	agent.Steps = nil
}
