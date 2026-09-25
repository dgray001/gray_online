package risq

import (
	"fmt"
	"os"

	"github.com/dgray001/gray_online/game"
	"github.com/dgray001/gray_online/game/games/risq/ai"
	"github.com/gin-gonic/gin"
)

func runAi(p *RisqPlayer, r *GameRisq, action_channel chan game.PlayerAction) {
	fmt.Println("Starting ai for AI player", p.player.GetAiId())
loop:
	for {
		if p.player.GetBase() == nil || p.player.GetBase().GameEnded() {
			break
		}
		select {
		case <-p.ai_stop:
			break loop
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
	decision := p.ai_model.DecideOrders(newAiView(p, r))
	for _, b := range decision.Behaviors {
		action_channel <- game.PlayerAction{Kind: "set-unit-behavior", Ai_id: int(p.player.GetAiId()), Action: behaviorAction(b)}
	}
	for _, b := range decision.BuildingBehaviors {
		action_channel <- game.PlayerAction{Kind: "set-building-behavior", Ai_id: int(p.player.GetAiId()), Action: buildingBehaviorAction(b)}
	}
	orders := make([]OrderFromFrontend, len(decision.Orders))
	for i, o := range decision.Orders {
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

func behaviorAction(b ai.UnitBehavior) gin.H {
	action := gin.H{"internal_ids": b.Subjects}
	if b.Stance != nil {
		action["stance"] = uint8(*b.Stance)
	}
	if b.InterruptCurrent != nil {
		action["interrupt_current"] = *b.InterruptCurrent
	}
	if b.AttackBack != nil {
		action["attack_back"] = *b.AttackBack
	}
	if b.TargetPriority != nil {
		action["target_priority"] = targetPriorityPayload(b.TargetPriority)
	}
	return action
}

func buildingBehaviorAction(b ai.BuildingBehavior) gin.H {
	action := gin.H{"internal_ids": b.Subjects}
	if b.AutoAttack != nil {
		action["auto_attack"] = *b.AutoAttack
	}
	if b.InterruptCurrent != nil {
		action["interrupt_current"] = *b.InterruptCurrent
	}
	if b.TargetPriority != nil {
		action["target_priority"] = targetPriorityPayload(b.TargetPriority)
	}
	return action
}

func targetPriorityPayload(priority []ai.TargetCategory) []uint8 {
	payload := make([]uint8, len(priority))
	for i, c := range priority {
		payload[i] = uint8(c)
	}
	return payload
}
