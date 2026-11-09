package rl

import (
	"fiddlesticks.live/neural"
	"github.com/dgray001/gray_online/game/games/fiddlesticks"
	"testing"
)

func TestRewardUsesExactBidEvenWhenPointsAreZero(t *testing.T) {
	for _, exact := range []bool{true, false} {
		n := initialNetwork(trial("zero", 0.1, 0, 1))
		input, legal := make([]float64, neural.Inputs), []int{0, 1}
		a := &Agent{Network: n, Steps: []neural.PolicyStep{{Input: input, Legal: legal, Action: 0, Probabilities: []float64{0.5, 0.5}}}}
		outcome := fiddlesticks.RoundOutcome{Round: 1, Bid: 0, Tricks: 0, Points: 0}
		if !exact {
			outcome.Tricks = 1
		}
		l := Learner{Rate: 0.1, Baselines: map[Context]Baseline{}}
		l.complete(a, outcome, 2)
		if (n.Predict(input, legal)[0] > 0.5) != exact || len(a.Steps) != 0 {
			t.Fatal("wrong reward direction or trajectory not cleared")
		}
	}
}
