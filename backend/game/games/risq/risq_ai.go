package risq

import (
	"fmt"
	"os"

	"github.com/dgray001/gray_online/game"
	"github.com/gin-gonic/gin"
)

func runAi(p *RisqPlayer, r *GameRisq, action_channel chan game.PlayerAction) {
	fmt.Println("Starting ai for AI player", p.player.GetAiId())
	for {
		if p.player.GetBase() == nil || p.player.GetBase().GameEnded() {
			break
		}
		select {
		case update := <-p.player.AiUpdates:
			if !p.player.GetBase().GameStarted() || p.player.GetBase().GameEnded() {
				break
			}
			p.ai_model.ApplyUpdate(newAiView(p, r), update.Kind)
			if update.Kind == "start-turn" {
				submitAiOrders(p, r, action_channel)
			}
		case <-p.player.AiFailedUpdates:
			fmt.Fprintln(os.Stderr, "AI player", p.player.GetAiId(), "received failed update")
		}
	}
	fmt.Println("Ending ai for AI player", p.player.GetAiId())
}

func submitAiOrders(p *RisqPlayer, r *GameRisq, action_channel chan game.PlayerAction) {
	ai_orders := p.ai_model.DecideOrders(newAiView(p, r))
	orders := make([]OrderFromFrontend, len(ai_orders))
	for i, o := range ai_orders {
		orders[i] = OrderFromFrontend{
			Player_id:             p.player.Player_id,
			Subjects:              o.Subjects,
			Order_type:            o.OrderType,
			Target_id:             o.TargetID,
			Clear_previous_orders: o.ClearPreviousOrders,
		}
	}
	action_channel <- game.PlayerAction{Kind: "submit-orders", Ai_id: int(p.player.GetAiId()), Action: gin.H{"orders": orders}}
}
