package neural

import "github.com/dgray001/gray_online/game/games/fiddlesticks"

func publicFeatures(input []float64, state fiddlesticks.LearningState, betting bool) []float64 {
	input[160] = float64(state.Round) / 26
	input[368] = float64(state.RoundPoints) / 99
	input[369] = float64(state.TrickPoints) / 99
	input[161] = float64(len(state.Bets)) / 51
	input[162] = float64((state.Dealer-state.Seat+len(state.Bets))%len(state.Bets)) / float64(len(state.Bets))
	if betting {
		input[163] = 1
	}
	for relative := range state.Bets {
		seat := (state.Seat + relative) % len(state.Bets)
		offset := 164 + 4*relative
		if state.HasBet[seat] {
			input[offset] = float64(state.Bets[seat]) / 26
			input[offset+3] = 1
		}
		input[offset+1] = float64(state.Tricks[seat]) / 26
		input[offset+2] = float64(state.Scores[seat]) / 1000
	}
	return input
}
