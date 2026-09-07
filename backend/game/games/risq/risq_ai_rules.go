package risq

import (
	"fmt"

	"github.com/dgray001/gray_online/game"
)

type RisqAiRule struct {
	when RisqAiCondition
	then []RisqAiAction
}

type RisqAiModelRules struct {
	rules []RisqAiRule
}

func (m *RisqAiModelRules) ApplyUpdate(p *RisqPlayer, r *GameRisq, update *game.UpdateMessage) {}

func (m *RisqAiModelRules) DecideOrders(p *RisqPlayer, r *GameRisq) []OrderFromFrontend {
	orders := make([]OrderFromFrontend, 0)
	for _, rule := range m.rules {
		if !rule.when.evaluate(p, r) {
			continue
		}
		for _, action := range rule.then {
			orders = append(orders, action.toOrders(p, r)...)
		}
	}
	return orders
}

func parseRisqAiRules(raw []interface{}) ([]RisqAiRule, error) {
	rules := make([]RisqAiRule, 0, len(raw))
	for _, item := range raw {
		obj, ok := item.(map[string]interface{})
		if !ok {
			return nil, fmt.Errorf("rule must be an object")
		}
		when_raw, ok := obj["when"].(map[string]interface{})
		if !ok {
			return nil, fmt.Errorf("rule missing \"when\" object")
		}
		when, err := parseRisqAiCondition(when_raw)
		if err != nil {
			return nil, err
		}
		then_raw, ok := obj["then"].([]interface{})
		if !ok {
			return nil, fmt.Errorf("rule missing \"then\" list")
		}
		then := make([]RisqAiAction, 0, len(then_raw))
		for _, action_raw := range then_raw {
			action_obj, ok := action_raw.(map[string]interface{})
			if !ok {
				return nil, fmt.Errorf("action must be an object")
			}
			action, err := parseRisqAiAction(action_obj)
			if err != nil {
				return nil, err
			}
			then = append(then, action)
		}
		rules = append(rules, RisqAiRule{when: when, then: then})
	}
	return rules, nil
}
