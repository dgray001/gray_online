package ai

func (c *issueContext) unitEligible(v verb, u UnitView, target *aim) bool {
	if v.name == "delete" || v.name == "ungarrison" {
		return true
	}
	if target == nil {
		return false
	}
	switch v.name {
	case "move":
		return true
	case "attack":
		return u.CanAttack && (target.unit != nil || target.building != nil || target.space || target.zone)
	case "garrison":
		return target.building != nil
	}
	return u.Type == UnitTypeEconomic && c.workEligible(v, u, target)
}

func (c *issueContext) eligible(v verb, actor loopItem, target *aim) bool {
	switch a := unwrapItem(actor).(type) {
	case unitItem:
		return c.unitEligible(v, a.unit, target)
	case buildingItem:
		return v.name == "delete" || v.name == "attack" && a.building.CanAttack && target != nil && (target.unit != nil || target.building != nil)
	case mercenaryItem:
		return v.name == "hire" && a.producible.ID == v.unit_id && target != nil && target.zone
	}
	return false
}

func (c *issueContext) buildEligible(id uint32, u UnitView, target *aim) bool {
	if f := target.foundation; f != nil {
		return (id == 0 || id == f.BuildingID) && canBuild(u, f.BuildingID) && c.view.BuildingAvailable(f.BuildingID)
	}
	return id != 0 && target.zone && canBuild(u, id) && c.view.BuildingAvailable(id) && c.view.ValidBuildZone(id, target.loc)
}

func (c *issueContext) workEligible(v verb, u UnitView, target *aim) bool {
	switch v.name {
	case "build":
		return c.buildEligible(v.building_id, u, target)
	case "gather":
		return target.resource != nil || target.building != nil || len(c.gatherSourcesIn(target)) > 0
	case "repair", "renew":
		return target.building != nil
	}
	return false
}
