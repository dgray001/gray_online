package ai

import "fmt"

type verb struct {
	name              string
	building_id       uint32
	unit_id           uint32
	tech_id           uint32
	count             amount
	queue             *amount
	to                string
	unit_behavior     *UnitBehavior
	building_behavior *BuildingBehavior
	stop              *stopSpec
}

var verbNames = []string{"move", "attack", "build", "gather", "garrison", "ungarrison", "repair", "renew", "delete", "hire", "cancel_foundation", "create", "research", "set_behavior", "stop"}

func (v verb) control() bool { return v.name == "set_behavior" || v.name == "stop" }

func (v verb) actorless() bool { return v.name == "hire" || v.name == "cancel_foundation" }

func (v verb) production() bool {
	return v.name == "create" || v.name == "research"
}

func parseVerbs(raw any, actor_kind string) ([]verb, error) {
	list, ok := raw.([]any)
	if !ok || len(list) == 0 {
		return nil, fmt.Errorf("\"issue\" must be a nonempty list of verbs")
	}
	verbs := make([]verb, 0, len(list))
	for _, entry := range list {
		obj, ok := entry.(map[string]any)
		if !ok || len(obj) != 1 {
			return nil, fmt.Errorf("each \"issue\" entry must be an object with one verb")
		}
		for name, params := range obj {
			v, err := parseVerb(name, params, actor_kind)
			if err != nil {
				return nil, err
			}
			verbs = append(verbs, v)
		}
	}
	return verbs, nil
}

func parseVerb(name string, params any, actor_kind string) (verb, error) {
	known := false
	for _, n := range verbNames {
		known = known || n == name
	}
	options, ok := params.(map[string]any)
	if !known || !ok {
		return verb{}, fmt.Errorf("unknown verb %q (or its options are not an object)", name)
	}
	v := verb{name: name}
	if v.control() {
		return parseControlVerb(v, options, actor_kind)
	}
	allowed := []string{}
	if name == "build" {
		allowed = append(allowed, "building_id")
		if _, present := options["building_id"]; present {
			id := parseOptionalID(options, "building_id")
			if id == nil {
				return verb{}, fmt.Errorf("build needs a whole-number \"building_id\"")
			}
			v.building_id = *id
		}
	}
	if name == "move" || name == "attack" {
		allowed = append(allowed, "to")
		if raw, present := options["to"]; present {
			if v.to, _ = raw.(string); v.to != "space" && v.to != "zone" {
				return verb{}, fmt.Errorf("%s \"to\" must be \"space\" or \"zone\"", name)
			}
		}
	}
	if v.production() {
		id_key := map[string]string{"create": "unit_id", "research": "tech_id"}[name]
		allowed = append(allowed, id_key, "queue")
		id := parseOptionalID(options, id_key)
		if id == nil {
			return verb{}, fmt.Errorf("%s needs a whole-number %q", name, id_key)
		}
		if name == "create" {
			v.unit_id = *id
		} else {
			v.tech_id = *id
		}
		if _, present := options["queue"]; present {
			queue, err := parseNumber(options, "queue", 0)
			if err != nil {
				return verb{}, fmt.Errorf("%s queue: %v", name, err)
			}
			v.queue = &queue
		}
	}
	if name == "hire" {
		allowed = append(allowed, "unit_id", "count")
		id := parseOptionalID(options, "unit_id")
		if id == nil {
			return verb{}, fmt.Errorf("hire needs a whole-number \"unit_id\"")
		}
		v.unit_id = *id
		var count amount
		err := withLoopScope("item", "select_mercenaries", func() (parsed error) {
			count, parsed = parseNumber(options, "count", 1)
			return parsed
		})
		if err != nil {
			return verb{}, fmt.Errorf("hire count: %v", err)
		}
		v.count = count
	}
	return v, checkKeys(options, allowed...)
}

type aim struct {
	unit       *UnitView
	building   *BuildingView
	resource   *ResourceView
	foundation *FoundationView
	loc        ZoneRef
	space      bool
	zone       bool
}

func aimAt(item loopItem) (aim, bool) {
	switch it := unwrapItem(item).(type) {
	case unitItem:
		return aim{unit: &it.unit, loc: it.loc}, true
	case buildingItem:
		return aim{building: &it.building, loc: it.loc}, true
	case resourceItem:
		return aim{resource: &it.resource, loc: it.loc}, true
	case zoneItem:
		return aim{loc: it.loc, zone: true}, true
	case spaceItem:
		return aim{loc: ZoneRef{Space: it.loc.Space}, space: true}, true
	case foundationItem:
		return aim{foundation: &it.foundation, loc: it.loc, zone: true}, true
	case gatherableItem:
		return gatherableAim(it), true
	case attackableItem:
		return attackableAim(it), true
	}
	return aim{}, false
}

func (a aim) contains(loc ZoneRef) bool {
	if a.space {
		return loc.Space == a.loc.Space
	}
	return a.zone && loc == a.loc
}

type issueContext struct {
	view           View
	internals      *Internals
	gather_sources map[aim][]ResourceView
	excluded       func(*targetSlot) bool
}

func (c *issueContext) issue(v verb, actor loopItem, target *aim, clear bool) (Order, bool) {
	if !c.eligible(v, actor, target) {
		return Order{}, false
	}
	switch a := unwrapItem(actor).(type) {
	case unitItem:
		return c.unitIssue(v, a.unit, target, clear)
	case buildingItem:
		return c.buildingIssue(v, a.building, target)
	case mercenaryItem:
		c.internals.spend(a.producible.Cost)
		c.internals.pending_population++
		return c.view.Issue(IssueRequest{Verb: VerbHire, Item: a.producible.ID, Target: IssueTarget{Zone: target.loc}}), true
	}
	return Order{}, false
}

func (c *issueContext) buildingIssue(v verb, b BuildingView, target *aim) (Order, bool) {
	switch {
	case v.name == "delete":
		return c.view.Issue(IssueRequest{Verb: VerbDeleteBuilding, Building: b}), true
	case target.unit != nil:
		return c.view.Issue(IssueRequest{Verb: VerbBuildingAttackUnit, Building: b, Target: IssueTarget{Unit: target.unit}}), true
	case target.building != nil:
		return c.view.Issue(IssueRequest{Verb: VerbBuildingAttackBuilding, Building: b, Target: IssueTarget{Building: target.building}}), true
	}
	return Order{}, false
}
