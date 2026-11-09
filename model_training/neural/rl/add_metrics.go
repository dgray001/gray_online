package rl

import "slices"

func addEpisode(m Metrics, episode Episode) Metrics {
	s, seat := episode.Scenario.Settings, episode.Scenario.Seat
	m.Games++
	m.Decisions += episode.Decisions
	m.Rounds += len(episode.Outcomes)
	m.OpponentExact += episode.OpponentExact
	m.OpponentRounds += episode.OpponentRounds
	for _, outcome := range episode.Outcomes {
		if outcome.Bid == outcome.Tricks {
			m.Exact++
		}
		if outcome.Bid == 0 {
			m.ZeroBids++
		}
		m.MeanBid += float64(outcome.Bid)
	}
	leaders, maximum, opponents := 0, slices.Max(episode.Scores), 0.0
	for i, score := range episode.Scores {
		if score == maximum {
			leaders++
		}
		if i != seat {
			opponents += float64(score)
		}
	}
	if episode.Scores[seat] == maximum {
		m.WinShare += 1 / float64(leaders)
	}
	rounds := float64(s.MaxRound)
	available := float64(s.RoundPoints)*(2*rounds-1) + float64(s.TrickPoints)*rounds*rounds
	m.Points += float64(episode.Scores[seat])
	m.OpponentPoints += opponents / float64(s.Players-1)
	m.OpponentScore += opponents / float64(s.Players-1) / available
	m.Score += float64(episode.Scores[seat]) / available
	m.Margin += (float64(episode.Scores[seat]) - opponents/float64(s.Players-1)) / available
	return m
}
