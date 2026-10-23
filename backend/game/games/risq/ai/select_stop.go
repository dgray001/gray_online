package ai

import "fmt"

type stopSpec struct {
	orders   map[uint8]bool
	item_ids map[uint32]bool
}

type stopOrder struct {
	id   uint64
	kind uint8
	item uint32
}

func (c *issueContext) stopOrders(s *stopSpec, actor loopItem) []Order {
	orders := make([]Order, 0)
	for _, active := range c.activeStopOrders(actor) {
		if (len(s.orders) > 0 && !s.orders[active.kind]) || (len(s.item_ids) > 0 && !s.item_ids[active.item]) {
			continue
		}
		orders = append(orders, c.cancelStopOrder(active.id))
	}
	return orders
}

func (c *issueContext) activeStopOrders(actor loopItem) []stopOrder {
	orders := make([]stopOrder, 0)
	switch a := unwrapItem(actor).(type) {
	case unitItem:
		for _, unit := range c.view.Units() {
			if unit.InternalID == a.unit.InternalID {
				for _, order := range unit.ActiveOrders {
					orders = append(orders, stopOrder{id: order.ID, kind: uint8(order.Kind)})
				}
			}
		}
	case buildingItem:
		building, _ := freshBuilding(c.view, a.building.InternalID)
		for _, order := range building.ActiveOrders {
			orders = append(orders, stopOrder{id: order.ID, kind: uint8(order.Kind), item: order.ItemID})
		}
	}
	return orders
}

func (c *issueContext) cancelStopOrder(id uint64) Order {
	for _, building := range c.view.Buildings() {
		for _, order := range building.ActiveOrders {
			if order.ID != id {
				continue
			}
			var cost Cost
			switch order.Kind {
			case BuildingOrderCreate:
				cost = c.view.UnitCost(order.ItemID)
			case BuildingOrderResearch:
				cost = c.view.TechCost(order.ItemID)
			}
			c.internals.spend(Cost{Food: -cost.Food, Wood: -cost.Wood, Stone: -cost.Stone, Gold: -cost.Gold})
		}
	}
	return c.view.Issue(IssueRequest{Verb: VerbCancelOrder, OrderID: id})
}

func parseStopSpec(options map[string]any, actor_kind string) (*stopSpec, error) {
	allowed := []string{"orders"}
	if actor_kind == "buildings" {
		allowed = append(allowed, "item_ids")
	}
	if err := checkKeys(options, allowed...); err != nil {
		return nil, err
	}
	return parseStopFilters(options, actor_kind)
}

func parseStopFilters(options map[string]any, actor_kind string) (*stopSpec, error) {
	s := &stopSpec{}
	var err error
	if actor_kind == "units" {
		s.orders, err = parseStopKinds(options["orders"], orderKindNames)
	} else {
		s.orders, err = parseStopKinds(options["orders"], buildingOrderKindNames)
	}
	if err != nil {
		return nil, err
	}
	if actor_kind == "buildings" {
		s.item_ids, err = parseIDSet(options, "item_ids")
	}
	return s, err
}

func parseStopKinds[T ~uint8](raw any, names map[string]T) (map[uint8]bool, error) {
	if raw == nil {
		return nil, nil
	}
	list, ok := raw.([]any)
	if !ok {
		return nil, fmt.Errorf("stop orders must be an array of names")
	}
	kinds := make(map[uint8]bool, len(list))
	for _, entry := range list {
		name, ok := entry.(string)
		kind, known := names[name]
		if !ok || !known {
			return nil, fmt.Errorf("unknown stop order kind %v", entry)
		}
		kinds[uint8(kind)] = true
	}
	return kinds, nil
}
