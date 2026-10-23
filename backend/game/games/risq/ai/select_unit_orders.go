package ai

import "slices"

func (c *issueContext) do(verb Verb, u UnitView, target IssueTarget, clear bool) Order {
	return c.view.Issue(IssueRequest{Verb: verb, Unit: u, Target: target, Clear: clear})
}

func (c *issueContext) unitIssue(v verb, u UnitView, target *aim, clear bool) (Order, bool) {
	switch v.name {
	case "delete":
		return c.do(VerbDeleteUnit, u, IssueTarget{}, clear), true
	case "ungarrison":
		return c.do(VerbUngarrison, u, IssueTarget{}, clear), true
	}
	switch v.name {
	case "move":
		return c.move(v, u, target, clear), true
	case "attack":
		return c.attack(v, u, target, clear)
	case "build":
		return c.build(v.building_id, u, target, clear)
	case "gather":
		return c.gather(u, target, clear)
	case "garrison":
		return c.garrison(u, target, clear)
	case "repair", "renew":
		return c.maintain(v.name, u, target, clear)
	}
	return Order{}, false
}

func (v verb) wantsSpace(default_space bool) bool {
	switch v.to {
	case "space":
		return true
	case "zone":
		return false
	}
	return default_space
}

func (c *issueContext) move(v verb, u UnitView, target *aim, clear bool) Order {
	if v.wantsSpace(target.space) {
		return c.do(VerbMoveSpace, u, IssueTarget{Space: target.loc.Space}, clear)
	}
	return c.do(VerbMoveZone, u, IssueTarget{Zone: target.loc}, clear)
}

func (c *issueContext) attack(v verb, u UnitView, target *aim, clear bool) (Order, bool) {
	if v.to != "" && (target.unit != nil || target.building != nil || target.space || target.zone) {
		if v.wantsSpace(target.space) {
			return c.do(VerbAttackSpace, u, IssueTarget{Space: target.loc.Space}, clear), true
		}
		return c.do(VerbAttackZone, u, IssueTarget{Zone: target.loc}, clear), true
	}
	switch {
	case target.unit != nil:
		return c.do(VerbAttackUnit, u, IssueTarget{Unit: target.unit}, clear), true
	case target.building != nil:
		return c.do(VerbAttackBuilding, u, IssueTarget{Building: target.building}, clear), true
	case target.space:
		return c.do(VerbAttackSpace, u, IssueTarget{Space: target.loc.Space}, clear), true
	case target.zone:
		return c.do(VerbAttackZone, u, IssueTarget{Zone: target.loc}, clear), true
	}
	return Order{}, false
}

func (c *issueContext) garrison(u UnitView, target *aim, clear bool) (Order, bool) {
	return c.do(VerbGarrison, u, IssueTarget{Building: target.building}, clear), true
}

func (c *issueContext) maintain(name string, u UnitView, target *aim, clear bool) (Order, bool) {
	b := target.building
	if name == "repair" {
		return c.do(VerbRepair, u, IssueTarget{Building: b}, clear), true
	}
	if !b.Renewing && !c.internals.renewals[b.InternalID] {
		c.internals.renewals[b.InternalID] = true
		c.internals.spend(b.RenewCost)
	}
	return c.do(VerbRenew, u, IssueTarget{Building: b}, clear), true
}

func (c *issueContext) gather(u UnitView, target *aim, clear bool) (Order, bool) {
	if target.resource != nil {
		return c.do(VerbGather, u, IssueTarget{Resource: target.resource}, clear), true
	}
	if target.building != nil {
		source := gatherableAsResource(*target.building)
		return c.do(VerbGather, u, IssueTarget{Resource: &source}, clear), true
	}
	best, found := ResourceView{}, false
	for _, source := range c.gatherSourcesIn(target) {
		if !found || c.view.LocationDistance(u.Location, source.Location) < c.view.LocationDistance(u.Location, best.Location) {
			best, found = source, true
		}
	}
	if !found {
		return Order{}, false
	}
	return c.do(VerbGather, u, IssueTarget{Resource: &best}, clear), true
}

func gatherableAsResource(b BuildingView) ResourceView {
	return ResourceView{InternalID: b.InternalID, Category: b.ResourceCategory, Location: b.Location, AmountLeft: b.ResourcesLeft, GatherSpeed: b.GatherSpeed, GatherCapacity: b.GatherCapacity, IsBuilding: true}
}

func (c *issueContext) gatherSourcesIn(target *aim) []ResourceView {
	if sources, cached := c.gather_sources[*target]; cached {
		return sources
	}
	sources := make([]ResourceView, 0)
	for _, category := range []ResourceCategory{ResourceFood, ResourceWood, ResourceStone, ResourceGold} {
		for _, r := range c.view.KnownResources(category) {
			if target.contains(r.Location) {
				sources = append(sources, r)
			}
		}
	}
	for _, b := range c.view.AllBuildings() {
		if target.contains(b.Location) {
			sources = append(sources, gatherableAsResource(b))
		}
	}
	if c.gather_sources == nil {
		c.gather_sources = make(map[aim][]ResourceView)
	}
	c.gather_sources[*target] = sources
	return sources
}

func (c *issueContext) assistFoundation(u UnitView, f *FoundationView, clear bool) (Order, bool) {
	return c.view.Issue(IssueRequest{Verb: VerbBuild, Unit: u, Item: f.BuildingID, Target: IssueTarget{Zone: f.Location}, Clear: clear}), true
}

func (c *issueContext) build(building_id uint32, u UnitView, target *aim, clear bool) (Order, bool) {
	if target.foundation != nil {
		return c.assistFoundation(u, target.foundation, clear)
	}
	if !slices.ContainsFunc(c.view.Foundations(), func(f FoundationView) bool { return f.Location == target.loc }) {
		c.internals.spend(c.view.BuildCost(building_id))
	}
	request := IssueRequest{Verb: VerbBuild, Unit: u, Item: building_id, Target: IssueTarget{Zone: target.loc}, Clear: clear}
	return c.view.Issue(request), true
}
