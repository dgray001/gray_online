package ai

import (
	"encoding/json"
	"fmt"

	"github.com/dgray001/gray_online/util"
)

type Rule struct {
	when Condition
	then []Action
}

type RulesModel struct {
	rules []Rule
}

func (m *RulesModel) ApplyUpdate(view View, update_kind string) {
	if update_kind == "start-turn" {
		current, limit := view.Population()
		data, _ := json.Marshal(map[string]interface{}{
			"population": map[string]int{"current": current, "limit": limit},
			"resources": map[string]float64{
				"food":  view.Resource(ResourceFood),
				"wood":  view.Resource(ResourceWood),
				"stone": view.Resource(ResourceStone),
				"gold":  view.Resource(ResourceGold),
			},
			"units":           view.Units(),
			"buildings":       view.Buildings(),
			"enemy_units":     view.VisibleEnemyUnits(),
			"enemy_buildings": view.VisibleEnemyBuildings(),
		})
		util.DebugLog.Printf("ai %s: new turn data %s", view.Nickname(), data)
	}
}

func (m *RulesModel) DecideOrders(view View) []Order {
	orders := make([]Order, 0)
	for _, rule := range m.rules {
		if !rule.when.Evaluate(view) {
			continue
		}
		for _, action := range rule.then {
			orders = append(orders, action.ToOrders(view)...)
		}
	}
	util.DebugLog.Printf("ai %s: submitting orders %+v", view.Nickname(), orders)
	return orders
}

func parseRules(raw []interface{}) ([]Rule, error) {
	rules := make([]Rule, 0, len(raw))
	for _, item := range raw {
		obj, ok := item.(map[string]interface{})
		if !ok {
			return nil, fmt.Errorf("rule must be an object")
		}
		when_raw, ok := obj["when"].(map[string]interface{})
		if !ok {
			return nil, fmt.Errorf("rule missing \"when\" object")
		}
		when, err := parseCondition(when_raw)
		if err != nil {
			return nil, err
		}
		then_raw, ok := obj["then"].([]interface{})
		if !ok {
			return nil, fmt.Errorf("rule missing \"then\" list")
		}
		then := make([]Action, 0, len(then_raw))
		for _, action_raw := range then_raw {
			action_obj, ok := action_raw.(map[string]interface{})
			if !ok {
				return nil, fmt.Errorf("action must be an object")
			}
			action, err := parseAction(action_obj)
			if err != nil {
				return nil, err
			}
			then = append(then, action)
		}
		rules = append(rules, Rule{when: when, then: then})
	}
	return rules, nil
}
