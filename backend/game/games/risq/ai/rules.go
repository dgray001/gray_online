package ai

import (
	"encoding/json"

	"github.com/dgray001/gray_online/util"
)

type QKind uint8

const (
	QUnit QKind = iota
	QBuilding
	QTech
	QResource
)

type Q struct {
	Cost   Cost
	ID     *uint32
	Type   QKind
	Weight float64
}

type Bucket struct {
	Desired int
	Members map[uint64]bool
	Task    Action
}

type Internals struct {
	q       []Q
	Buckets map[string]*Bucket
}

func (i *Internals) Refresh() {
	i.q = make([]Q, 0)
}

func (i *Internals) bucket(name string) *Bucket {
	if i.Buckets == nil {
		i.Buckets = make(map[string]*Bucket)
	}
	b, ok := i.Buckets[name]
	if !ok {
		b = &Bucket{Members: make(map[uint64]bool)}
		i.Buckets[name] = b
	}
	return b
}

func (i *Internals) isBucketed(unit_id uint64) bool {
	for _, b := range i.Buckets {
		if b.Members[unit_id] {
			return true
		}
	}
	return false
}

func (i *Internals) pruneBuckets(view View) {
	if len(i.Buckets) == 0 {
		return
	}
	alive := make(map[uint64]bool)
	for _, u := range view.Units() {
		alive[u.InternalID] = true
	}
	for _, b := range i.Buckets {
		for id := range b.Members {
			if !alive[id] {
				delete(b.Members, id)
			}
		}
	}
}

type Rule struct {
	when Condition
	then []Action
}

type RulesModel struct {
	rules     []Rule
	internals Internals
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
	m.internals.Refresh()
	m.internals.pruneBuckets(view)
	orders := make([]Order, 0)
	for _, rule := range m.rules {
		if !rule.when.Evaluate(view) {
			continue
		}
		for _, action := range rule.then {
			orders = append(orders, action.ToOrders(view, &m.internals)...)
		}
	}
	util.DebugLog.Printf("ai %s: submitting orders %+v", view.Nickname(), orders)
	return orders
}
