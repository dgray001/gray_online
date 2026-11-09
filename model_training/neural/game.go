package neural

import (
	"github.com/dgray001/gray_online/game"
	"github.com/dgray001/gray_online/game/games/fiddlesticks"
)

func Play(network *Network, examples *[]Example, settings GameSettings, seat int) []uint32 {
	base := game.CreateBaseGame(1, game.GameType_FIDDLESTICKS, nil, nil, nil)
	f := fiddlesticks.InitializeGame(base)
	for i := 0; i < settings.Players; i++ {
		model := uint8(4)
		if len(settings.OpponentModels) > 0 && i != seat {
			model = settings.OpponentModels[i%len(settings.OpponentModels)]
		}
		f.AddInternalAiPlayer(model, nil)
	}
	for i := 0; i < settings.Players; i++ {
		p := f.GetFiddlesticksPlayer(i)
		agent := network
		if len(settings.Networks) > 0 {
			agent = settings.Networks[i]
		}
		if (examples != nil && (i == seat || len(settings.OpponentModels) == 0)) || (agent != nil && (i == seat || len(settings.Networks) > 0)) {
			p.SetAiModel(&Player{Network: agent, Teacher: p.AiModel(), Examples: examples})
		}
	}
	f.SetSettings(1, settings.MaxRound, settings.RoundPoints, settings.TrickPoints, 1)
	if settings.DealSeed != 0 {
		f.SetDealSeed(settings.DealSeed)
	}
	base.StartGame()
	f.StartAiGame(settings.DealSeed == 0)
	for !f.ExecuteAiTurn(false) {
	}
	_, scores := f.GetGameResults()
	return scores
}
