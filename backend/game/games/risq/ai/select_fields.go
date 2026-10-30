package ai

import (
	"fmt"
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
	rank_keys []float64
	role      string
	lets      *letState
	cache     map[string]float64
	capacity  *float64
	initial   *float64
	slot      *targetSlot
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

func (s selItem) letValue(name string, spec amount) float64 {
	if val, ok := s.cachedLet(name); ok {
		return val
	}
	if s.lets.busy[name] {
		s.internals.warnOnce("let:"+name, "ai: let "+name+" depends on itself")
		s.internals.eval_errors++
		return 0
	}
	if s.lets.busy == nil {
		s.lets.busy = map[string]bool{}
	}
	s.lets.busy[name] = true
	restore := s.internals.bindItem(s.role, s)
	errors_before := s.internals.eval_errors
	value := spec.float(s.view, s.internals)
	restore()
	s.lets.busy[name] = false
	s.cacheLet(name, value, errors_before)
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
	case orderItem:
		return "orders"
	case assignmentItem:
		return "assignments"
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
	if value, ok, nested := s.pathField(name); nested {
		return value, ok
	}
	if s.role == "actor" && name == "assign_round" {
		return s.internals.assign_round, true
	}
	if s.role == "target" {
		switch name {
		case "assigned":
			if s.slot != nil {
				return s.slot.assigned, true
			}
			if s.initial != nil {
				return *s.initial, true
			}
			return 0, true
		case "remaining":
			if s.slot != nil && s.slot.remaining != nil {
				return *s.slot.remaining, true
			}
			return float64(unboundedCount), true
		case "already_targeted":
			return s.alreadyTargeted(), true
		}
	}
	if get, ok := selectFieldTables[kindOfItem(s.loopItem)][name]; ok {
		return get(s), true
	}
	value, ok := s.loopItem.field(name)
	if !ok {
		if _, strict := unwrapItem(s.loopItem).(assignmentItem); strict {
			s.internals.warnOnce("item:"+s.role+"."+name, fmt.Sprintf("ai: var(%s.%s) is not readable for this assignment", s.role, name))
			s.internals.eval_errors++
		}
	}
	return value, ok
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

func (s selItem) alreadyTargeted() float64 {
	actorItem := s.internals.loop_items["actor"]
	tk, hasT := keyOf(s.loopItem)
	ak, hasA := keyOf(actorItem)
	if !hasT || !hasA || tk.id == 0 || ak.id == 0 {
		return 0
	}
	if u, ok := unwrapItem(actorItem).(unitItem); ok && orderTargetUID(u.unit.CurrentOrder) == float64(tk.id) {
		return 1
	}
	if s.internals.hasQueuedTarget(s.view, ak.id, tk.id) {
		return 1
	}
	return 0
}

func (i *Internals) hasQueuedTarget(view View, subjectUID uint64, targetUID uint64) bool {
	if i.target_queue_index == nil {
		i.target_queue_index = make(map[uint64]map[uint64]bool)
		if view != nil {
			for _, entry := range view.OrderQueue() {
				if entry.SubjectUID != 0 && entry.TargetUID != 0 {
					if i.target_queue_index[entry.SubjectUID] == nil {
						i.target_queue_index[entry.SubjectUID] = make(map[uint64]bool)
					}
					i.target_queue_index[entry.SubjectUID][entry.TargetUID] = true
				}
			}
		}
	}
	return i.target_queue_index[subjectUID][targetUID]
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

var selectSeenFields = fieldTable[selItem]{
	"turns_since_seen": func(s selItem) float64 {
		loc, _, _ := locOf(s.loopItem)
		return float64(s.view.TurnNumber() - s.internals.last_observed[loc.Space])
	},
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
	"orders":      {},
	"assignments": {},
	"units":       mergeFields(selectLocatedFields, selectSeenFields, selectUnitFields),
	"buildings":   mergeFields(selectLocatedFields, selectSeenFields, selectBuildingFields),
	"resources":   mergeFields(selectLocatedFields, selectSeenFields),
	"spaces":      mergeFields(selectLocatedFields, selectSeenFields, selectEmptyField),
	"zones":       mergeFields(selectLocatedFields, selectSeenFields, selectEmptyField),
	"regions":     {},

	"foundations": mergeFields(selectLocatedFields, selectSeenFields),
	"gatherables": mergeFields(selectLocatedFields, selectSeenFields),
	"attackables": mergeFields(selectLocatedFields, selectSeenFields),
	"mercenaries": selectMercenaryFields,
}

func init() {
	for kind, table := range selectFieldTables {
		all := slices.Concat(itemFields[kind], table.names(), pathFieldNames(kind))
		itemFields["select_"+kind] = all
		itemFields["select_target_"+kind] = slices.DeleteFunc(slices.Clone(all), func(n string) bool {
			return n == "distance_to_actor" || n == "zone_distance_to_actor"
		})
		itemFields["select_actor_"+kind] = slices.DeleteFunc(slices.Clone(all), func(n string) bool {
			return n == "distance_to_target" || n == "zone_distance_to_target"
		})
	}
}
