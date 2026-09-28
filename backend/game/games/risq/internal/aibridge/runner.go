package aibridge

import (
	"fmt"
	"os"

	"github.com/dgray001/gray_online/game"
	"github.com/dgray001/gray_online/game/games/risq/ai"
	"github.com/dgray001/gray_online/game/games/risq/internal/defs"
	"github.com/gin-gonic/gin"
)

type Runner struct {
	Player  *game.Player
	Model   ai.Model
	Stop    <-chan struct{}
	Actions chan game.PlayerAction
}

func (a *Runner) Run() {
	fmt.Println("Starting ai for AI player", a.Player.GetAiId())
	fallback_submitted := false
loop:
	for {
		if a.Player.GetBase() == nil || a.Player.GetBase().GameEnded() {
			break
		}
		select {
		case <-a.Stop:
			break loop
		case update := <-a.Player.AiUpdates:
			if !a.Player.GetBase().GameStarted() || a.Player.GetBase().GameEnded() || update.Kind != "start-turn" {
				break
			}
			fallback_submitted = false
			a.decideAndSubmit(update)
		case update := <-a.Player.AiFailedUpdates:
			fmt.Fprintln(os.Stderr, "AI player", a.Player.GetAiId(), "received failed update", update.Kind, update.Content["message"])
			// a rejected batch would otherwise leave the turn waiting on this AI forever
			if update.Kind == "submit-orders-failed" && !fallback_submitted {
				fallback_submitted = true
				a.send("submit-orders", gin.H{"orders": []defs.OrderFromFrontend{}})
			}
		}
	}
	fmt.Println("Ending ai for AI player", a.Player.GetAiId())
}

// Decides from the same start-turn payload a human client receives; a human's own submission wins over the ai's.
func (a *Runner) decideAndSubmit(update *game.UpdateMessage) {
	payload, _ := update.Content["game"].(gin.H)
	snapshot, err := parseAiSnapshot(payload)
	var view *aiView
	if err == nil {
		view, _ = newAiView(snapshot, a.Player.Player_id)
	}
	if view == nil {
		fmt.Fprintln(os.Stderr, "AI player", a.Player.GetAiId(), "could not read start-turn payload:", err)
		a.send("submit-orders", gin.H{"orders": []defs.OrderFromFrontend{}})
		return
	}
	a.Model.ApplyUpdate(view, update.Kind)
	decision := a.Model.DecideOrders(view)
	if a.Player.IsHumanPlayer() {
		return
	}
	for _, b := range decision.Behaviors {
		a.send("set-unit-behavior", behaviorAction(b))
	}
	for _, b := range decision.BuildingBehaviors {
		a.send("set-building-behavior", buildingBehaviorAction(b))
	}
	orders := make([]defs.OrderFromFrontend, len(decision.Orders))
	for i, o := range decision.Orders {
		orders[i] = defs.OrderFromFrontend{
			Player_id:             a.Player.Player_id,
			Subjects:              o.Subjects,
			Order_type:            o.OrderType,
			Target_id:             o.TargetID,
			Clear_previous_orders: o.ClearPreviousOrders,
		}
	}
	a.send("submit-orders", gin.H{"orders": orders})
}

// Gives up once the ai is stopped so a full channel nobody drains anymore can't strand this goroutine
func (a *Runner) send(kind string, action gin.H) {
	select {
	case a.Actions <- game.PlayerAction{Kind: kind, Ai_id: int(a.Player.GetAiId()), Action: action}:
	case <-a.Stop:
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
