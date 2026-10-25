package ai

import "fmt"

func findProducible(b BuildingView, kind ProducibleKind, id uint32) (Producible, bool) {
	for _, p := range b.Producibles {
		if p.Kind == kind && p.ID == id {
			return p, true
		}
	}
	return Producible{}, false
}

func freshBuilding(view View, internal_id uint64) (BuildingView, bool) {
	for _, b := range view.Buildings() {
		if b.InternalID == internal_id {
			return b, true
		}
	}
	return BuildingView{}, false
}

func (a *selectAction) parseProduction(raw map[string]any) error {
	for _, v := range a.verbs {
		if !v.production() {
			return fmt.Errorf("select: create and research cannot be mixed with other verbs")
		}
	}
	if raw["actors"] == nil || raw["match"] != nil || raw["mode"] != nil || raw["pick"] != nil {
		return fmt.Errorf("select: create and research need explicit \"actors\" and take no \"pick\", \"match\" or \"mode\"")
	}
	if a.actors.kind != "buildings" {
		return fmt.Errorf("select: create and research need \"buildings\" actors")
	}
	return nil
}

func (s *selectorSpec) takeProduction(where map[string]any) error {
	for key, target := range map[string]*uint32{"can_create": &s.can_create, "can_research": &s.can_research} {
		if _, present := where[key]; !present {
			continue
		}
		id := parseOptionalID(where, key)
		if id == nil || s.kind != "buildings" {
			return fmt.Errorf("%q needs a whole-number id and only filters buildings", key)
		}
		*target = *id
		delete(where, key)
	}
	if _, present := where["can_build"]; present {
		id := parseOptionalID(where, "can_build")
		if id == nil || s.kind != "units" {
			return fmt.Errorf("%q needs a whole-number id and only filters units", "can_build")
		}
		s.can_build = *id
		delete(where, "can_build")
	}
	return nil
}

func (s *selectorSpec) offers(item loopItem) bool {
	switch it := unwrapItem(item).(type) {
	case buildingItem:
		if (s.can_create != 0 || s.can_research != 0) && it.building.UnderConstruction {
			return false
		}
		_, creates := findProducible(it.building, ProducibleUnit, s.can_create)
		_, researches := findProducible(it.building, ProducibleTech, s.can_research)
		return (s.can_create == 0 || creates) && (s.can_research == 0 || researches)
	case unitItem:
		return s.can_build == 0 || canBuild(it.unit, s.can_build)
	}
	return true
}

// One order, or enough to bring the building's production queue up to the verb's depth
func (c *issueContext) productionCount(v verb, actor loopItem, b BuildingView) (int, bool) {
	if v.queue == nil {
		return 1, true
	}
	selected := actor.(selItem)
	building := selected.loopItem.(buildingItem)
	building.building = b
	selected.loopItem = building
	defer c.internals.bindItem("actor", selected)()
	previous := c.internals.eval_errors
	depth := v.queue.int(c.view, c.internals)
	return depth - productionOrderCount(b), c.internals.eval_errors == previous
}

func (c *issueContext) produce(v verb, actor loopItem) []Order {
	chosen, ok := unwrapItem(actor).(buildingItem)
	if !ok {
		return nil
	}
	b, found := freshBuilding(c.view, chosen.building.InternalID)
	kind, id, issue_verb := ProducibleUnit, v.unit_id, VerbCreateUnit
	if v.name == "research" {
		kind, id, issue_verb = ProducibleTech, v.tech_id, VerbResearch
	}
	item, offered := findProducible(b, kind, id)
	if !found || b.UnderConstruction || !offered {
		return nil
	}
	count, valid := c.productionCount(v, actor, b)
	if !valid {
		return nil
	}
	if kind == ProducibleTech {
		count = min(count, 1)
	}
	orders := make([]Order, 0, max(count, 0))
	for range max(count, 0) {
		c.internals.spend(item.Cost)
		if kind == ProducibleUnit {
			c.internals.pending_population++
		}
		orders = append(orders, c.view.Issue(IssueRequest{Verb: issue_verb, Building: b, Item: id}))
	}
	return orders
}
