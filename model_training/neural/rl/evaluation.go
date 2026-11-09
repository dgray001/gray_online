package rl

import "fmt"

func summarize(name string, episodes []Episode) Evaluation {
	result := Evaluation{Name: name, Groups: map[string]Metrics{}}
	for _, episode := range episodes {
		s := episode.Scenario.Settings
		keys := []string{"all", fmt.Sprintf("players=%d", s.Players), fmt.Sprintf("scoring=%d:%d", s.RoundPoints, s.TrickPoints), "opponents=" + episode.Scenario.Opponents, fmt.Sprintf("rounds=%d", s.MaxRound)}
		if s.Players >= 4 && s.Players <= 8 {
			keys = append(keys, "players=4-8")
		} else {
			keys = append(keys, "players=fringe")
		}
		recordHands(&result, episode)
		for _, key := range keys {
			result.Groups[key] = addEpisode(result.Groups[key], episode)
		}
	}
	for key, m := range result.Groups {
		m.ExactRate = float64(m.Exact) / float64(m.Rounds)
		m.OpponentExactRate = float64(m.OpponentExact) / float64(m.OpponentRounds)
		m.ZeroBidRate = float64(m.ZeroBids) / float64(m.Rounds)
		m.MeanBid /= float64(m.Rounds)
		m.WinShare /= float64(m.Games)
		m.Margin /= float64(m.Games)
		m.Score /= float64(m.Games)
		m.Points /= float64(m.Games)
		m.OpponentPoints /= float64(m.Games)
		m.OpponentScore /= float64(m.Games)
		result.Groups[key] = m
	}
	result.All = result.Groups["all"]
	finishHands(&result)
	return result
}
