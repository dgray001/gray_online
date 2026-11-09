package goldset

import (
	"github.com/dgray001/gray_online/game"
	"github.com/dgray001/gray_online/game/games/fiddlesticks"
)

func generateGame(q GenerateRequest, players int, gameIndex int, pools map[string]*reservoir) {
	seed := q.Seed*1000 + int64(gameIndex)
	source := Source{Seed: seed, ActionSeed: seed + 7919, Players: players, RoundPoints: q.RoundPoints, TrickPoints: q.TrickPoints}
	base := game.CreateBaseGame(uint64(gameIndex), game.GameType_FIDDLESTICKS, nil, nil, nil)
	f := fiddlesticks.InitializeGame(base)
	for i := 0; i < players; i++ {
		f.AddInternalAiPlayer(4, nil)
	}
	decision := 0
	for i := 0; i < players; i++ {
		p := f.GetFiddlesticksPlayer(i)
		model := p.AiModel().(*fiddlesticks.FiddlesticksAiModelTheory4)
		p.SetAiModel(&collector{Model: model, Capture: func(p *fiddlesticks.FiddlesticksPlayer, f *fiddlesticks.GameFiddlesticks) *Item {
			decision++
			s := source
			s.Decision = decision
			return capture(q, s, p, f, model, pools)
		}, Accept: func(item Item) { pools[item.State.Kind].Items[item.slot] = item }})
	}
	f.SetSettings(1, uint8(51/players), q.RoundPoints, q.TrickPoints, 1)
	f.SetDealSeed(source.Seed)
	f.SetActionSeed(source.ActionSeed)
	base.StartGame()
	f.StartAiGame(false)
	for !f.ExecuteAiTurn(false) {
	}
}
