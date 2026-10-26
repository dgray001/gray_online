package ai

// What is being built at a zone: a foundation, or one of our buildings still under construction
func buildingAt(view View, zone ZoneRef) (uint32, bool) {
	for _, f := range view.Foundations() {
		if f.Location == zone {
			return f.BuildingID, true
		}
	}
	for _, b := range view.Buildings() {
		if b.Location == zone && b.UnderConstruction {
			return b.BuildingID, true
		}
	}
	return 0, false
}

func (c *issueContext) requeue(actor loopItem) (Order, bool) {
	unit, ok := unwrapItem(actor).(unitItem)
	if !ok || unit.unit.CurrentOrder == nil {
		return Order{}, false
	}
	previous := unit.unit.CurrentOrder
	request := IssueRequest{Unit: unit.unit}
	switch {
	case previous.Kind == OrderKindMove && previous.TargetZone != nil:
		request.Verb, request.Target.Zone = VerbMoveZone, *previous.TargetZone
	case previous.Kind == OrderKindMove && previous.TargetSpace != nil:
		request.Verb, request.Target.Space = VerbMoveSpace, *previous.TargetSpace
	case previous.Kind == OrderKindGather && previous.TargetResource != nil:
		request.Verb, request.Target.Resource = VerbGather, previous.TargetResource
	case previous.Kind == OrderKindRepair && previous.TargetBuilding != nil:
		request.Verb, request.Target.Building = VerbRepair, previous.TargetBuilding
	case previous.Kind == OrderKindRenew && previous.TargetBuilding != nil:
		request.Verb, request.Target.Building = VerbRenew, previous.TargetBuilding
	case previous.Kind == OrderKindGarrison && previous.TargetBuilding != nil:
		request.Verb, request.Target.Building = VerbGarrison, previous.TargetBuilding
	case previous.Kind == OrderKindAttackUnit && previous.TargetUnit != nil:
		request.Verb, request.Target.Unit = VerbAttackUnit, previous.TargetUnit
	case previous.Kind == OrderKindAttackBuilding && previous.TargetBuilding != nil:
		request.Verb, request.Target.Building = VerbAttackBuilding, previous.TargetBuilding
	case previous.Kind == OrderKindAttackSpace && previous.TargetSpace != nil:
		request.Verb, request.Target.Space = VerbAttackSpace, *previous.TargetSpace
	case previous.Kind == OrderKindAttackZone && previous.TargetZone != nil:
		request.Verb, request.Target.Zone = VerbAttackZone, *previous.TargetZone
	case previous.Kind == OrderKindBuild && previous.TargetZone != nil:
		building_id, found := buildingAt(c.view, *previous.TargetZone)
		if !found {
			return Order{}, false
		}
		request.Verb, request.Item, request.Target.Zone = VerbBuild, building_id, *previous.TargetZone
	default:
		return Order{}, false
	}
	return c.view.Issue(request), true
}
