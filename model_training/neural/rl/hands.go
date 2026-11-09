package rl

import "github.com/dgray001/gray_online/game/games/fiddlesticks"

type RoundMetrics struct {
	Rounds      int
	Exact       int
	ZeroBids    int
	ExactRate   float64
	ZeroBidRate float64
}

func addHand(m RoundMetrics, outcome fiddlesticks.RoundOutcome) RoundMetrics {
	m.Rounds++
	if outcome.Bid == outcome.Tricks {
		m.Exact++
	}
	if outcome.Bid == 0 {
		m.ZeroBids++
	}
	return m
}

func recordHands(e *Evaluation, episode Episode) {
	if e.Hands == nil {
		e.Hands, e.CoreHands = map[uint8]RoundMetrics{}, map[uint8]RoundMetrics{}
	}
	core := episode.Scenario.Settings.Players >= 4 && episode.Scenario.Settings.Players <= 8
	for _, outcome := range episode.Outcomes {
		e.Hands[outcome.Round] = addHand(e.Hands[outcome.Round], outcome)
		if core {
			e.CoreHands[outcome.Round] = addHand(e.CoreHands[outcome.Round], outcome)
		}
		if outcome.Round >= (episode.Scenario.Settings.MaxRound+1)/2 {
			e.Upper = addHand(e.Upper, outcome)
			if core {
				e.CoreUpper = addHand(e.CoreUpper, outcome)
			}
		}
	}
}
