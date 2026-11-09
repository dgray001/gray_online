package experiment

import (
	"fiddlesticks.live/neural"
	"testing"
)

func TestSummariesKeepAgentsSeparate(t *testing.T) {
	scenario := Scenario{Settings: neural.GameSettings{Players: 2, MaxRound: 1, RoundPoints: 10}, Seat: 0}
	a := scoredGame("a", scenario, []uint32{10, 0})
	b := scoredGame("b", scenario, []uint32{0, 10})
	for _, summary := range summarize([]GameResult{a, b}) {
		if summary.Group != "all" {
			continue
		}
		if summary.Games != 1 || (summary.Agent == "a" && summary.WinShare != 1) || (summary.Agent == "b" && summary.WinShare != 0) {
			t.Fatal(summary)
		}
	}
}
