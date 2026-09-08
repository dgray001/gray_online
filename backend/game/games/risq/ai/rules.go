package ai

import (
	"encoding/json"

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
