package experiment

import (
	"fiddlesticks.live/neural"
	"slices"
)

type GameResult struct {
	Agent      string
	Settings   neural.GameSettings
	Opponents  string
	ActionSeed int64
	Seat       int
	Scores     []uint32
	WinShare   float64
	Chance     float64
	Score      float64
	Margin     float64
}

func scoredGame(name string, scenario Scenario, scores []uint32) GameResult {
	settings, seat := scenario.Settings, scenario.Seat
	leaders, maximum, opponents := 0, slices.Max(scores), 0.0
	for i, score := range scores {
		if score == maximum {
			leaders++
		}
		if i != seat {
			opponents += float64(score)
		}
	}
	rounds := float64(settings.MaxRound)
	available := float64(settings.RoundPoints)*(2*rounds-1) + float64(settings.TrickPoints)*rounds*rounds
	result := GameResult{Agent: name, Settings: settings, Opponents: scenario.Opponents, ActionSeed: scenario.ActionSeed, Seat: seat, Scores: scores, Chance: 1 / float64(settings.Players)}
	if scores[seat] == maximum {
		result.WinShare = 1 / float64(leaders)
	}
	result.Score = float64(scores[seat]) / available
	result.Margin = (float64(scores[seat]) - opponents/float64(settings.Players-1)) / available
	return result
}
