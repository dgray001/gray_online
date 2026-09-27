package risq

import (
	"fmt"
	"os"

	"github.com/dgray001/gray_online/game"
	"github.com/dgray001/gray_online/game/games/risq/ai"
	"github.com/gin-gonic/gin"
)

func runAi(p *RisqPlayer, action_channel chan game.PlayerAction) {
	fmt.Println("Starting ai for AI player", p.player.GetAiId())
	fallback_submitted := false
loop:
	for {
		if p.player.GetBase() == nil || p.player.GetBase().GameEnded() {
			break
		}
		select {
		case <-p.ai_stop:
			break loop
		case update := <-p.player.AiUpdates:
			if !p.player.GetBase().GameStarted() || p.player.GetBase().GameEnded() || update.Kind != "start-turn" {
				break
			}
			fallback_submitted = false
			decideAndSubmit(p, update, action_channel)
		case update := <-p.player.AiFailedUpdates:
			fmt.Fprintln(os.Stderr, "AI player", p.player.GetAiId(), "received failed update", update.Kind, update.Content["message"])
			// a rejected batch would otherwise leave the turn waiting on this AI forever
			if update.Kind == "submit-orders-failed" && !fallback_submitted {
				fallback_submitted = true
				sendAiAction(p, action_channel, "submit-orders", gin.H{"orders": []OrderFromFrontend{}})
			}
		}
	}
	fmt.Println("Ending ai for AI player", p.player.GetAiId())
}

// Decides from the same start-turn payload a human client receives; a human's own submission wins over the ai's.
func decideAndSubmit(p *RisqPlayer, update *game.UpdateMessage, action_channel chan game.PlayerAction) {
	payload, _ := update.Content["game"].(gin.H)
	snapshot, err := parseAiSnapshot(payload)
	var view *aiView
	if err == nil {
		view, _ = newAiView(snapshot, p.player.Player_id)
	}
	if view == nil {
		fmt.Fprintln(os.Stderr, "AI player", p.player.GetAiId(), "could not read start-turn payload:", err)
		sendAiAction(p, action_channel, "submit-orders", gin.H{"orders": []OrderFromFrontend{}})
		return
	}
	p.ai_model.ApplyUpdate(view, update.Kind)
	decision := p.ai_model.DecideOrders(view)
	if p.player.IsHumanPlayer() {
		return
	}
	for _, b := range decision.Behaviors {
		sendAiAction(p, action_channel, "set-unit-behavior", behaviorAction(b))
	}
	for _, b := range decision.BuildingBehaviors {
		sendAiAction(p, action_channel, "set-building-behavior", buildingBehaviorAction(b))
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
	sendAiAction(p, action_channel, "submit-orders", gin.H{"orders": orders})
}

// Gives up once the ai is stopped so a full channel nobody drains anymore can't strand this goroutine
func sendAiAction(p *RisqPlayer, action_channel chan game.PlayerAction, kind string, action gin.H) {
	select {
	case action_channel <- game.PlayerAction{Kind: kind, Ai_id: int(p.player.GetAiId()), Action: action}:
	case <-p.ai_stop:
	}
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
