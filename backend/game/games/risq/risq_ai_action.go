package risq

import "fmt"

type RisqAiAction interface {
	toOrders(p *RisqPlayer, r *GameRisq) []OrderFromFrontend
}

func parseRisqAiAction(raw map[string]interface{}) (RisqAiAction, error) {
	action_type, ok := raw["action"].(string)
	if !ok {
		return nil, fmt.Errorf("action must have a string \"action\" field")
	}
	switch action_type {
	default:
		return nil, fmt.Errorf("unknown action type %q", action_type)
	}
}
