package ai

import (
	"maps"
	"slices"
)

type fieldTable[T any] map[string]func(T) float64

func (t fieldTable[T]) get(item T, name string) (float64, bool) {
	getter, ok := t[name]
	if !ok {
		return 0, false
	}
	return getter(item), true
}

func (t fieldTable[T]) names() []string {
	return slices.Sorted(maps.Keys(t))
}

func withFields[T any](own fieldTable[T], base fieldTable[place], at func(T) place) fieldTable[T] {
	merged := maps.Clone(own)
	for name, get := range base {
		merged[name] = func(item T) float64 { return get(at(item)) }
	}
	return merged
}

func withPlace[T any](own fieldTable[T], at func(T) place) fieldTable[T] {
	return withFields(own, placeFields, at)
}

func withLocation[T any](own fieldTable[T], at func(T) place) fieldTable[T] {
	return withFields(own, locationFields, at)
}

func orderKind(o *CurrentOrder) float64 {
	if o == nil {
		return -1
	}
	return float64(o.Kind)
}

func orderTargetUID(o *CurrentOrder) float64 {
	if o != nil && o.TargetUID != 0 {
		return float64(o.TargetUID)
	}
	switch {
	case o == nil:
	case o.TargetUnit != nil:
		return float64(unitUID(o.TargetUnit.InternalID))
	case o.TargetBuilding != nil:
		return float64(buildingUID(o.TargetBuilding.InternalID))
	case o.TargetResource != nil:
		return float64(o.TargetResource.uid())
	}
	return 0
}

func orderTargetLocation(o *CurrentOrder) ZoneRef {
	switch {
	case o == nil:
	case o.TargetZone != nil:
		return *o.TargetZone
	case o.TargetSpace != nil:
		return ZoneRef{Space: *o.TargetSpace}
	case o.TargetResource != nil:
		return o.TargetResource.Location
	case o.TargetUnit != nil:
		return o.TargetUnit.Location
	case o.TargetBuilding != nil:
		return o.TargetBuilding.Location
	}
	return ZoneRef{}
}

var locationFields = fieldTable[place]{
	"x":      func(p place) float64 { return float64(p.loc.Space.X) },
	"y":      func(p place) float64 { return float64(p.loc.Space.Y) },
	"zone_x": func(p place) float64 { return float64(p.loc.Zone.X) },
	"zone_y": func(p place) float64 { return float64(p.loc.Zone.Y) },
}

var placeFields = func() fieldTable[place] {
	fields := maps.Clone(locationFields)
	fields["distance_home"] = func(p place) float64 { return float64(p.distance_home) }
	return fields
}()

var playerFields = fieldTable[playerItem]{
	"id":              func(p playerItem) float64 { return float64(p.id) },
	"is_me":           func(p playerItem) float64 { return boolNumber(p.me) },
	"is_eliminated":   func(p playerItem) float64 { return boolNumber(p.eliminated) },
	"score":           func(p playerItem) float64 { return float64(p.score) },
	"units_visible":   func(p playerItem) float64 { return float64(p.units_visible) },
	"buildings_known": func(p playerItem) float64 { return float64(p.buildings_known) },
}

var spaceFields = withPlace(fieldTable[spaceItem]{
	"terrain_type":         func(s spaceItem) float64 { return float64(s.info.TerrainType) },
	"intra_cost":           func(s spaceItem) float64 { return float64(s.info.IntraCost) },
	"inter_cost":           func(s spaceItem) float64 { return float64(s.info.InterCost) },
	"build_speed_modifier": func(s spaceItem) float64 { return s.info.BuildSpeedModifier },
	"passable":             func(s spaceItem) float64 { return boolNumber(s.info.Passable) },
	"vision":               func(s spaceItem) float64 { return float64(s.vision) },
	"owner":                func(s spaceItem) float64 { return float64(s.owner) },
	"unit_count":           func(s spaceItem) float64 { return float64(s.unit_count) },
	"unidentified_count":   func(s spaceItem) float64 { return float64(s.unidentified_count) },
	"gold_income":          func(s spaceItem) float64 { return s.info.GoldIncome },
}, func(s spaceItem) place { return s.place })

var regionFields = fieldTable[regionItem]{
	"index":          func(r regionItem) float64 { return float64(r.stat.index) },
	"gold_bonus":     func(r regionItem) float64 { return r.stat.info.GoldBonus },
	"owner":          func(r regionItem) float64 { return float64(r.stat.info.Owner) },
	"known_spaces":   func(r regionItem) float64 { return float64(len(r.stat.info.Spaces)) },
	"owned_spaces":   func(r regionItem) float64 { return float64(r.stat.owned) },
	"enemy_spaces":   func(r regionItem) float64 { return float64(r.stat.enemy) },
	"unowned_spaces": func(r regionItem) float64 { return float64(r.stat.unowned) },
	"has_unexplored": func(r regionItem) float64 { return boolNumber(r.stat.info.HasUnexplored) },
}

var mercenaryFields = fieldTable[mercenaryItem]{
	"id":           func(m mercenaryItem) float64 { return float64(m.producible.ID) },
	"kind":         func(m mercenaryItem) float64 { return float64(m.producible.Kind) },
	"unit_type":    func(m mercenaryItem) float64 { return float64(m.producible.UnitType) },
	"cost_food":    func(m mercenaryItem) float64 { return m.producible.Cost.Food },
	"cost_wood":    func(m mercenaryItem) float64 { return m.producible.Cost.Wood },
	"cost_stone":   func(m mercenaryItem) float64 { return m.producible.Cost.Stone },
	"cost_gold":    func(m mercenaryItem) float64 { return m.producible.Cost.Gold },
	"stamina_cost": func(m mercenaryItem) float64 { return float64(m.producible.StaminaCost) },
}

var unitFields = withPlace(fieldTable[unitItem]{
	"uid":               func(u unitItem) float64 { return float64(unitUID(u.unit.InternalID)) },
	"player":            func(u unitItem) float64 { return float64(u.unit.PlayerID) },
	"unit_id":           func(u unitItem) float64 { return float64(u.unit.UnitID) },
	"unit_type":         func(u unitItem) float64 { return float64(u.unit.Type) },
	"is_military":       func(u unitItem) float64 { return boolNumber(u.unit.Kind == UnitMilitary) },
	"is_garrisoned":     func(u unitItem) float64 { return boolNumber(u.unit.GarrisonedIn != nil) },
	"health":            func(u unitItem) float64 { return u.unit.Health },
	"max_health":        func(u unitItem) float64 { return u.unit.MaxHealth },
	"stamina":           func(u unitItem) float64 { return float64(u.unit.CurrentStamina) },
	"can_attack":        func(u unitItem) float64 { return boolNumber(u.unit.CanAttack) },
	"max_stamina":       func(u unitItem) float64 { return float64(u.unit.MaxStamina) },
	"attack_range":      func(u unitItem) float64 { return float64(u.unit.AttackRange) },
	"attack_reach":      func(u unitItem) float64 { return float64(u.unit.AttackRange.SpaceReach()) },
	"has_order":         func(u unitItem) float64 { return boolNumber(u.unit.CurrentOrder != nil) },
	"order_kind":        func(u unitItem) float64 { return orderKind(u.unit.CurrentOrder) },
	"order_target_uid":  func(u unitItem) float64 { return orderTargetUID(u.unit.CurrentOrder) },
	"order_space_x":     func(u unitItem) float64 { return float64(orderTargetLocation(u.unit.CurrentOrder).Space.X) },
	"order_space_y":     func(u unitItem) float64 { return float64(orderTargetLocation(u.unit.CurrentOrder).Space.Y) },
	"order_zone_x":      func(u unitItem) float64 { return float64(orderTargetLocation(u.unit.CurrentOrder).Zone.X) },
	"order_zone_y":      func(u unitItem) float64 { return float64(orderTargetLocation(u.unit.CurrentOrder).Zone.Y) },
	"stance":            func(u unitItem) float64 { return float64(u.unit.Stance) },
	"attack_back":       func(u unitItem) float64 { return boolNumber(u.unit.AttackBack) },
	"interrupt_current": func(u unitItem) float64 { return boolNumber(u.unit.InterruptCurrent) },
}, func(u unitItem) place { return u.place })

var buildingFields = withPlace(fieldTable[buildingItem]{
	"uid":                   func(b buildingItem) float64 { return float64(buildingUID(b.building.InternalID)) },
	"player":                func(b buildingItem) float64 { return float64(b.building.PlayerID) },
	"building_id":           func(b buildingItem) float64 { return float64(b.building.BuildingID) },
	"is_under_construction": func(b buildingItem) float64 { return boolNumber(b.building.UnderConstruction) },
	"health":                func(b buildingItem) float64 { return b.building.Health },
	"max_health":            func(b buildingItem) float64 { return b.building.MaxHealth },
	"max_stamina":           func(b buildingItem) float64 { return float64(b.building.MaxStamina) },
	"attack_range":          func(b buildingItem) float64 { return float64(b.building.AttackRange) },
	"attack_reach":          func(b buildingItem) float64 { return float64(b.building.AttackRange.SpaceReach()) },
	"is_visible":            func(b buildingItem) float64 { return boolNumber(b.building.Visible) },
	"garrison_count":        func(b buildingItem) float64 { return float64(b.building.GarrisonCount) },
	"garrison_capacity":     func(b buildingItem) float64 { return float64(b.building.GarrisonCapacity) },
	"resources_left":        func(b buildingItem) float64 { return b.building.ResourcesLeft },
	"gather_speed":          func(b buildingItem) float64 { return float64(b.building.GatherSpeed) },
	"gather_capacity":       func(b buildingItem) float64 { return float64(b.building.GatherCapacity) },
	"resource_category":     func(b buildingItem) float64 { return float64(b.building.ResourceCategory) },
	"is_renewing":           func(b buildingItem) float64 { return boolNumber(b.building.Renewing) },
	"is_gatherable":         func(b buildingItem) float64 { return boolNumber(b.building.Gatherable) },
	"renew_cost_food":       func(b buildingItem) float64 { return b.building.RenewCost.Food },
	"renew_cost_wood":       func(b buildingItem) float64 { return b.building.RenewCost.Wood },
	"renew_cost_stone":      func(b buildingItem) float64 { return b.building.RenewCost.Stone },
	"renew_cost_gold":       func(b buildingItem) float64 { return b.building.RenewCost.Gold },
	"can_attack":            func(b buildingItem) float64 { return boolNumber(b.building.CanAttack) },
}, func(b buildingItem) place { return b.place })

var resourceFields = withPlace(fieldTable[resourceItem]{
	"uid":             func(r resourceItem) float64 { return float64(r.resource.uid()) },
	"category":        func(r resourceItem) float64 { return float64(r.resource.Category) },
	"amount_left":     func(r resourceItem) float64 { return r.resource.AmountLeft },
	"gather_speed":    func(r resourceItem) float64 { return float64(r.resource.GatherSpeed) },
	"gather_capacity": func(r resourceItem) float64 { return float64(r.resource.GatherCapacity) },
}, func(r resourceItem) place { return r.place })

var foundationFields = withLocation(fieldTable[foundationItem]{
	"uid":               func(f foundationItem) float64 { return float64(f.uid) },
	"building_id":       func(f foundationItem) float64 { return float64(f.foundation.BuildingID) },
	"is_planned":        func(f foundationItem) float64 { return boolNumber(f.foundation.Planned) },
	"builders":          func(f foundationItem) float64 { return float64(f.foundation.Builders) },
	"ordered_this_turn": func(f foundationItem) float64 { return boolNumber(f.foundation.OrderedThisTurn) },
}, func(f foundationItem) place { return f.place })

var zoneFields = withPlace(fieldTable[zoneItem]{
	"has_resource":    func(z zoneItem) float64 { return boolNumber(z.has_resource) },
	"building_player": func(z zoneItem) float64 { return float64(z.building_player) },
}, func(z zoneItem) place { return z.place })
