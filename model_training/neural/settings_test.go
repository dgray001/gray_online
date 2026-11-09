package neural

import (
	"github.com/dgray001/gray_online/game/games/fiddlesticks"
	"testing"
)

func TestScoringInputs(t *testing.T) {
	for _, scoring := range [][2]uint16{{10, 1}, {0, 5}, {7, 0}, {99, 99}} {
		state := fiddlesticks.LearningState{Bets: []uint8{0, 0}, Tricks: []uint8{0, 0}, Scores: []uint32{0, 0}, HasBet: []bool{false, false}, RoundPoints: scoring[0], TrickPoints: scoring[1]}
		input := publicFeatures(make([]float64, Inputs), state, true)
		if input[368] != float64(scoring[0])/99 || input[369] != float64(scoring[1])/99 {
			t.Fatal("incorrect scoring inputs", scoring)
		}
	}
}
