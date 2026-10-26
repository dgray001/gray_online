package aibridge

import (
	"encoding/json"
	"slices"

	"github.com/dgray001/gray_online/game/games/risq/ai"
)

type viewCollections struct {
	resources   map[ai.ResourceCategory][]ai.ResourceView
	spaces      []ai.SpaceInfo
	zones       []ai.ZoneInfo
	matching    map[string][]ai.Coordinate
	owned       revisionCollection[ai.BuildingView]
	buildings   revisionCollection[ai.BuildingView]
	foundations revisionCollection[ai.FoundationView]
	orders      revisionCollection[ai.OrderEntry]
}

type revisionCollection[T any] struct {
	revision uint64
	ready    bool
	items    []T
}

type keyedSpaceCondition struct {
	ai.SpaceCondition
	All []keyedSpaceCondition `json:"all"`
	Any []keyedSpaceCondition `json:"any"`
	Not *keyedSpaceCondition  `json:"not"`
}

func keySpaceCondition(condition ai.SpaceCondition) keyedSpaceCondition {
	key := keyedSpaceCondition{SpaceCondition: condition}
	if condition.Any != nil {
		key.Any = make([]keyedSpaceCondition, len(condition.Any))
		for i, child := range condition.Any {
			key.Any[i] = keySpaceCondition(child)
		}
	}
	if condition.Not != nil {
		child := keySpaceCondition(*condition.Not)
		key.Not = &child
	}
	if condition.All != nil {
		key.All = make([]keyedSpaceCondition, len(condition.All))
		for i, child := range condition.All {
			key.All[i] = keySpaceCondition(child)
		}
	}
	return key
}

func (c *revisionCollection[T]) load(revision uint64, build func() []T) []T {
	if !c.ready || c.revision != revision {
		c.items = build()
		c.revision, c.ready = revision, true
	}
	return slices.Clone(c.items)
}

func (v *aiView) KnownResources(category ai.ResourceCategory) []ai.ResourceView {
	if v.collections.resources == nil {
		v.collections.resources = make(map[ai.ResourceCategory][]ai.ResourceView)
		for _, entry := range v.zones {
			if r := entry.zone.Resource; r != nil && r.ResourcesLeft > 0 {
				kind := toAiCategory(v.config.Resources[r.ResourceId].Category)
				v.collections.resources[kind] = append(v.collections.resources[kind], v.resourceNode(r, entry.ref))
			}
		}
	}
	resources := v.collections.resources[category]
	if resources == nil {
		return make([]ai.ResourceView, 0)
	}
	return slices.Clone(resources)
}

func (v *aiView) AllSpaces() []ai.SpaceInfo {
	if v.collections.spaces == nil {
		v.collections.spaces = v.buildAllSpaces()
	}
	return slices.Clone(v.collections.spaces)
}

func (v *aiView) AllZones() []ai.ZoneInfo {
	if v.collections.zones == nil {
		v.collections.zones = v.buildAllZones()
	}
	return slices.Clone(v.collections.zones)
}

func (v *aiView) OrderQueue() []ai.OrderEntry {
	return v.collections.orders.load(v.revision, v.buildOrderQueue)
}

func (v *aiView) matchingSpaces(condition ai.SpaceCondition) []ai.Coordinate {
	encoded, err := json.Marshal(keySpaceCondition(condition))
	if err != nil {
		return v.buildMatchingSpaces(condition)
	}
	key := string(encoded)
	if matches, found := v.collections.matching[key]; found {
		return slices.Clone(matches)
	}
	matches := v.buildMatchingSpaces(condition)
	if v.collections.matching == nil {
		v.collections.matching = make(map[string][]ai.Coordinate)
	}
	v.collections.matching[key] = matches
	return slices.Clone(matches)
}

func (v *aiView) AllBuildings() []ai.BuildingView {
	buildings := v.collections.buildings.load(v.revision, v.buildAllBuildings)
	for i := range buildings {
		if buildings[i].PlayerID == v.playerId() {
			buildings[i].Producibles = slices.Clone(buildings[i].Producibles)
			buildings[i].ActiveOrders = slices.Clone(buildings[i].ActiveOrders)
			buildings[i].TargetPriority = slices.Clone(buildings[i].TargetPriority)
		}
	}
	return buildings
}
