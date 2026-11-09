package neural

import (
	"github.com/dgray001/gray_online/game/games/fiddlesticks"
	"math"
)

func historyFeatures(input []float64, state fiddlesticks.LearningState) {
	players := len(state.Bets)
	for _, event := range state.RoundHistory {
		slot := int(event.Suit-1)*13 + int(event.Number-2)
		relative := (event.Seat - state.Seat + players) % players
		angle := 2 * math.Pi * float64(relative) / float64(players)
		input[370+3*slot], input[371+3*slot] = math.Sin(angle), math.Cos(angle)
		input[372+3*slot] = float64((event.Trick-1)*players+event.Order+1) / float64(players*int(state.Round))
	}
	for relative := 0; relative < min(players, 10); relative++ {
		seat := (state.Seat + relative) % players
		if seat < len(state.Voids) {
			for suit, missing := range state.Voids[seat] {
				if missing {
					input[526+relative*4+suit] = 1
				}
			}
		}
		if seat < len(state.Profiles) {
			profileFeatures(input[566+8*relative:574+8*relative], state.Profiles[seat])
		}
		if seat < len(state.PriorProfiles) {
			profileFeatures(input[646+8*relative:654+8*relative], state.PriorProfiles[seat])
		}
	}
}
