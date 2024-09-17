package fiddlesticks

import "github.com/dgray001/gray_online/util"

/**
Random model just bets and plays cards randomly within what is legal
*/

type FiddlesticksAiModelRandom struct{}

func (m FiddlesticksAiModelRandom) Bet(p *FiddlesticksPlayer, f *GameFiddlesticks) int {
	return util.RandomInt(0, int(f.round))
}

func (m FiddlesticksAiModelRandom) PlayCard(p *FiddlesticksPlayer, f *GameFiddlesticks, valid_cards []int) int {
	return util.RandomInt(0, len(valid_cards)-1)
}
