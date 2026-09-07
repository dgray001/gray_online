package risq

import "github.com/dgray001/gray_online/game"

type RisqAiModel interface {
	ApplyUpdate(p *RisqPlayer, r *GameRisq, update *game.UpdateMessage)
	DecideOrders(p *RisqPlayer, r *GameRisq) []OrderFromFrontend
}

type RisqAiModelNoop struct{}

func (m *RisqAiModelNoop) ApplyUpdate(p *RisqPlayer, r *GameRisq, update *game.UpdateMessage) {}

func (m *RisqAiModelNoop) DecideOrders(p *RisqPlayer, r *GameRisq) []OrderFromFrontend {
	return []OrderFromFrontend{}
}
