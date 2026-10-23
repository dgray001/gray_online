package ai

import (
	"maps"
	"math"
	"slices"
)

type selItem struct {
	loopItem
	view      View
	internals *Internals
	census    *census
	rank      float64
	role      string
	lets      *letState
	cache     map[string]float64
}

func (s selItem) once(name string, compute func() float64) float64 {
	if value, done := s.cache[name]; done {
		return value
	}
	value := compute()
	if s.cache != nil {
		s.cache[name] = value
	}
	return value
}

type letState struct {
	specs map[string]amount
	busy  map[string]bool
}

func (s selItem) letValue(name string, spec amount) float64 {
	if s.lets.busy[name] {
		s.internals.warnOnce("let:"+name, "ai: let "+name+" depends on itself")
		s.internals.eval_errors++
		return 0
	}
	s.lets.busy[name] = true
	restore := s.internals.bindItem(s.role, s)
	value := spec.float(s.view, s.internals)
	restore()
	s.lets.busy[name] = false
	return value
}

func unwrapItem(item loopItem) loopItem {
	if s, ok := item.(selItem); ok {
		return s.loopItem
	}
	return item
}

func locOf(item loopItem) (ZoneRef, bool, bool) {
	switch it := unwrapItem(item).(type) {
	case unitItem:
		return it.loc, false, true
	case buildingItem:
		return it.loc, false, true
	case resourceItem:
		return it.loc, false, true
	case zoneItem:
		return it.loc, false, true
	case foundationItem:
		return it.loc, false, true
	case gatherableItem:
		return it.loc, false, true
	case attackableItem:
		return it.loc, false, true
	case spaceItem:
		return ZoneRef{Space: it.loc.Space}, true, true
	}
	return ZoneRef{}, false, false
}

func kindOfItem(item loopItem) string {
	switch unwrapItem(item).(type) {
	case unitItem:
		return "units"
	case buildingItem:
		return "buildings"
	case resourceItem:
		return "resources"
	case spaceItem:
		return "spaces"
	case zoneItem:
		return "zones"
	case regionItem:
		return "regions"
	case foundationItem:
		return "foundations"
	case mercenaryItem:
		return "mercenaries"
	case gatherableItem:
		return "gatherables"
	case attackableItem:
		return "attackables"
	}
	return ""
}

func (s selItem) field(name string) (float64, bool) {
	if s.lets != nil {
		if spec, ok := s.lets.specs[name]; ok {
			return s.letValue(name, spec), true
		}
	}
	if get, ok := selectFieldTables[kindOfItem(s.loopItem)][name]; ok {
		return get(s), true
	}
	if name == "distance_home" {
		return 0, false
	}
	return s.loopItem.field(name)
}

func (s selItem) here(count func(*census) tally) float64 {
	loc, whole_space, _ := locOf(s.loopItem)
	return float64(count(s.census).at(loc, whole_space))
}

func (s selItem) measure(a ZoneRef, b ZoneRef, zone_steps bool) float64 {
	if zone_steps {
		return float64(s.view.LocationDistance(a, b))
	}
	return float64(s.view.SpaceDistance(a.Space, b.Space))
}

func (s selItem) distanceToRole(role string, zone_steps bool) float64 {
	other, bound := s.internals.loop_items[role]
	if !bound {
		return 0
	}
	loc, _, _ := locOf(s.loopItem)
	other_loc, _, located := locOf(other)
	if !located {
		return 0
	}
	return s.measure(loc, other_loc, zone_steps)
}

func (s selItem) region(read func(regionStat) float64) float64 {
	loc, _, _ := locOf(s.loopItem)
	stat, _ := s.census.regions.at(loc.Space)
	return read(stat)
}

func (s selItem) spaceInfo(read func(SpaceInfo) float64) float64 {
	loc, _, _ := locOf(s.loopItem)
	return read(s.census.spaces[loc.Space])
}

var selectRegionFields = fieldTable[selItem]{
	"region":            func(s selItem) float64 { return s.region(func(r regionStat) float64 { return float64(r.index) }) },
	"region_owner":      func(s selItem) float64 { return s.region(func(r regionStat) float64 { return float64(r.info.Owner) }) },
	"region_gold_bonus": func(s selItem) float64 { return s.region(func(r regionStat) float64 { return r.info.GoldBonus }) },
	"region_known": func(s selItem) float64 {
		return s.region(func(r regionStat) float64 { return float64(len(r.info.Spaces)) })
	},
	"region_owned":   func(s selItem) float64 { return s.region(func(r regionStat) float64 { return float64(r.owned) }) },
	"region_enemy":   func(s selItem) float64 { return s.region(func(r regionStat) float64 { return float64(r.enemy) }) },
	"region_unowned": func(s selItem) float64 { return s.region(func(r regionStat) float64 { return float64(r.unowned) }) },
	"region_unexplored": func(s selItem) float64 {
		return s.region(func(r regionStat) float64 { return boolNumber(r.info.HasUnexplored) })
	},
	"turns_since_seen": func(s selItem) float64 {
		loc, _, _ := locOf(s.loopItem)
		return float64(s.view.TurnNumber() - s.internals.last_observed[loc.Space])
	},
	"space_vision":      func(s selItem) float64 { return s.spaceInfo(func(i SpaceInfo) float64 { return float64(i.Vision) }) },
	"space_owner":       func(s selItem) float64 { return s.spaceInfo(func(i SpaceInfo) float64 { return float64(i.Owner) }) },
	"space_gold_income": func(s selItem) float64 { return s.spaceInfo(func(i SpaceInfo) float64 { return i.GoldIncome }) },
}

var selectLocatedFields = fieldTable[selItem]{
	"distance_to_actor":       func(s selItem) float64 { return s.distanceToRole("actor", false) },
	"distance_to_target":      func(s selItem) float64 { return s.distanceToRole("target", false) },
	"distance_to_item":        func(s selItem) float64 { return s.distanceToRole("item", false) },
	"zone_distance_to_actor":  func(s selItem) float64 { return s.distanceToRole("actor", true) },
	"zone_distance_to_target": func(s selItem) float64 { return s.distanceToRole("target", true) },
	"zone_distance_to_item":   func(s selItem) float64 { return s.distanceToRole("item", true) },
	"own_units":               func(s selItem) float64 { return s.here(func(c *census) tally { return c.own_units }) },
	"enemy_units":             func(s selItem) float64 { return s.here(func(c *census) tally { return c.enemy_units }) },
	"own_buildings":           func(s selItem) float64 { return s.here(func(c *census) tally { return c.own_buildings }) },
	"enemy_buildings":         func(s selItem) float64 { return s.here(func(c *census) tally { return c.enemy_buildings }) },
	"resource_count":          func(s selItem) float64 { return s.here(func(c *census) tally { return c.resources }) },
}

var selectEmptyField = fieldTable[selItem]{
	"empty": func(s selItem) float64 {
		loc, whole_space, _ := locOf(s.loopItem)
		return boolNumber(s.census.total(loc, whole_space) == 0)
	},
}

var selectUnitFields = fieldTable[selItem]{
	"idle": func(s selItem) float64 {
		return boolNumber(s.census.idle_units[unwrapItem(s.loopItem).(unitItem).unit.InternalID])
	},
	"hp_pct": func(s selItem) float64 {
		u := unwrapItem(s.loopItem).(unitItem).unit
		return healthShare(u.Health, u.MaxHealth)
	},
}

var selectBuildingFields = fieldTable[selItem]{
	"idle": func(s selItem) float64 {
		return boolNumber(s.census.idle_buildings[unwrapItem(s.loopItem).(buildingItem).building.InternalID])
	},
	"hp_pct": func(s selItem) float64 {
		b := unwrapItem(s.loopItem).(buildingItem).building
		return healthShare(b.Health, b.MaxHealth)
	},
	"queue_length": func(s selItem) float64 {
		return float64(productionOrderCount(unwrapItem(s.loopItem).(buildingItem).building))
	},
}

const unboundedCount = math.MaxInt32

func maxAffordable(view View, internals *Internals, cost Cost) float64 {
	most := float64(unboundedCount)
	for _, category := range []ResourceCategory{ResourceFood, ResourceWood, ResourceStone, ResourceGold} {
		if price := cost.of(category); price > 0 {
			most = min(most, math.Floor(internals.available(view, category)/price))
		}
	}
	return max(0, most)
}

var selectMercenaryFields = fieldTable[selItem]{
	"can_afford": func(s selItem) float64 {
		return s.once("can_afford", func() float64 {
			return maxAffordable(s.view, s.internals, unwrapItem(s.loopItem).(mercenaryItem).producible.Cost)
		})
	},
}

func healthShare(health float64, max_health float64) float64 {
	if max_health <= 0 {
		return 1
	}
	return health / max_health
}

func mergeFields(tables ...fieldTable[selItem]) fieldTable[selItem] {
	merged := fieldTable[selItem]{}
	for _, t := range tables {
		maps.Copy(merged, t)
	}
	return merged
}

var selectFieldTables = map[string]fieldTable[selItem]{
	"units":     mergeFields(selectLocatedFields, selectRegionFields, selectUnitFields),
	"buildings": mergeFields(selectLocatedFields, selectRegionFields, selectBuildingFields),
	"resources": mergeFields(selectLocatedFields, selectRegionFields),
	"spaces":    mergeFields(selectLocatedFields, selectRegionFields, selectEmptyField),
	"zones":     mergeFields(selectLocatedFields, selectRegionFields, selectEmptyField),
	"regions":   {},

	"foundations": mergeFields(selectLocatedFields, selectRegionFields),
	"gatherables": mergeFields(selectLocatedFields, selectRegionFields),
	"attackables": mergeFields(selectLocatedFields, selectRegionFields),
	"mercenaries": selectMercenaryFields,
}

func init() {
	for kind, table := range selectFieldTables {
		inherited := slices.DeleteFunc(slices.Clone(itemFields[kind]), func(name string) bool { return name == "distance_home" })
		itemFields["select_"+kind] = append(inherited, table.names()...)
	}
}
