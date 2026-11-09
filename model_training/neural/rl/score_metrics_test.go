package rl

import (
	"fiddlesticks.live/neural"
	"fiddlesticks.live/neural/experiment"
	"github.com/dgray001/gray_online/game/games/fiddlesticks"
	"testing"
)

func TestScoreAveragesNormalizeEachGame(t *testing.T) {
	a := Episode{Scenario: experiment.Scenario{Settings: neural.GameSettings{Players: 2, MaxRound: 1, RoundPoints: 10, TrickPoints: 2}, Seat: 0}, Scores: []uint32{12, 0}, OpponentRounds: 1}
	b := Episode{Scenario: experiment.Scenario{Settings: neural.GameSettings{Players: 2, MaxRound: 1, RoundPoints: 20, TrickPoints: 4}, Seat: 0}, Scores: []uint32{0, 24}, OpponentRounds: 1}
	a.Outcomes = []fiddlesticks.RoundOutcome{{Round: 1, Bid: 1, Tricks: 1, Points: 12}}
	b.Outcomes = []fiddlesticks.RoundOutcome{{Round: 1, Bid: 1, Tricks: 0}}
	m := summarize("test", []Episode{a, b}).All
	if m.Points != 6 || m.OpponentPoints != 12 || m.Score != 0.5 || m.OpponentScore != 0.5 || m.Margin != 0 || m.WinShare != 0.5 {
		t.Fatal("incorrect mixed-setting score averages", m)
	}
}
