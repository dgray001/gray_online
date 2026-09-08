package ai

import "fmt"

func canAfford(view View, cost Cost) bool {
	return view.Resource(ResourceFood) >= cost.Food &&
		view.Resource(ResourceWood) >= cost.Wood &&
		view.Resource(ResourceStone) >= cost.Stone &&
		view.Resource(ResourceGold) >= cost.Gold
}

type gatherAction struct{ category ResourceCategory }

func (a *gatherAction) ToOrders(view View) []Order {
	orders := make([]Order, 0)
	for _, u := range view.IdleUnits() {
		if u.Kind != UnitEconomic {
			continue
		}
		if target, ok := view.NearestResource(u.Location, a.category); ok {
			orders = append(orders, view.GatherOrder(u, target, true))
		}
	}
	return orders
}

type produceAction struct{}

func (a *produceAction) ToOrders(view View) []Order {
	orders := make([]Order, 0)
	current, limit := view.Population()
	for _, b := range view.IdleBuildings() {
		for _, p := range b.Producibles {
			if p.Kind != ProducibleUnit || current >= limit || !canAfford(view, p.Cost) {
				continue
			}
			orders = append(orders, view.CreateUnitOrder(b, p.ID))
			current++
			break
		}
	}
	return orders
}

type researchAction struct{}

func (a *researchAction) ToOrders(view View) []Order {
	orders := make([]Order, 0)
	for _, b := range view.IdleBuildings() {
		for _, p := range b.Producibles {
			if p.Kind != ProducibleTech || !canAfford(view, p.Cost) {
				continue
			}
			orders = append(orders, view.ResearchOrder(b, p.ID))
			break
		}
	}
	return orders
}

// gate_pop_headroom, when set, skips building unless population is within max_pop_headroom of its cap.
// max_count, when set, skips building once the player already owns that many (0 means unlimited).
type buildAction struct {
	building_id       uint32
	gate_pop_headroom bool
	max_pop_headroom  int
	max_count         int
}

func countBuildingsOfType(view View, building_id uint32) int {
	count := 0
	for _, b := range view.Buildings() {
		if b.BuildingID == building_id {
			count++
		}
	}
	return count
}

func (a *buildAction) ToOrders(view View) []Order {
	orders := make([]Order, 0)
	if a.gate_pop_headroom {
		current, limit := view.Population()
		if limit-current > a.max_pop_headroom {
			return orders
		}
	}
	if a.max_count > 0 && countBuildingsOfType(view, a.building_id) >= a.max_count {
		return orders
	}
	if !canAfford(view, view.BuildCost(a.building_id)) {
		return orders
	}
	for _, u := range view.IdleUnits() {
		if u.Kind != UnitEconomic {
			continue
		}
		if target, ok := view.NearestBuildSite(u.Location, a.building_id); ok {
			orders = append(orders, view.BuildOrder(u, a.building_id, target, true))
		}
	}
	return orders
}

type exploreAction struct{}

func (a *exploreAction) ToOrders(view View) []Order {
	orders := make([]Order, 0)
	for _, u := range view.IdleUnits() {
		if target, ok := view.NearestUnexplored(u.Location); ok {
			orders = append(orders, view.MoveOrder(u, target, true))
		}
	}
	return orders
}

type attackTarget uint8

const (
	attackTargetMilitary attackTarget = iota // raiders vs. defenders: which enemy unit kind to prefer
	attackTargetEconomic
	attackTargetAny
)

type attackAction struct{ target attackTarget }

func (a *attackAction) ToOrders(view View) []Order {
	orders := make([]Order, 0)
	enemy_units := view.VisibleEnemyUnits()
	enemy_buildings := view.VisibleEnemyBuildings()
	for _, u := range view.IdleUnits() {
		if u.Kind != UnitMilitary {
			continue
		}
		if target, ok := a.pickTarget(u.Location, enemy_units); ok {
			orders = append(orders, view.AttackUnitOrder(u, target.InternalID, true))
		} else if target, ok := nearestBuilding(u.Location, enemy_buildings); ok {
			orders = append(orders, view.AttackBuildingOrder(u, target.InternalID, true))
		}
	}
	return orders
}

func (a *attackAction) pickTarget(from ZoneRef, units []UnitView) (UnitView, bool) {
	var preferred UnitKind
	switch a.target {
	case attackTargetEconomic:
		preferred = UnitEconomic
	case attackTargetAny:
		return nearestUnit(from, units)
	default:
		preferred = UnitMilitary
	}
	if target, ok := nearestUnitOfKind(from, units, preferred); ok {
		return target, true
	}
	return nearestUnit(from, units)
}

func nearestUnitOfKind(from ZoneRef, units []UnitView, kind UnitKind) (UnitView, bool) {
	filtered := make([]UnitView, 0, len(units))
	for _, u := range units {
		if u.Kind == kind {
			filtered = append(filtered, u)
		}
	}
	return nearestUnit(from, filtered)
}

func nearestUnit(from ZoneRef, units []UnitView) (UnitView, bool) {
	var best UnitView
	var best_distance int
	found := false
	for _, u := range units {
		d := locationDistance(from, u.Location)
		if !found || d < best_distance {
			best, best_distance, found = u, d, true
		}
	}
	return best, found
}

func nearestBuilding(from ZoneRef, buildings []BuildingView) (BuildingView, bool) {
	var best BuildingView
	var best_distance int
	found := false
	for _, b := range buildings {
		d := locationDistance(from, b.Location)
		if !found || d < best_distance {
			best, best_distance, found = b, d, true
		}
	}
	return best, found
}

func parseAction(raw map[string]interface{}) (Action, error) {
	action_type, ok := raw["action"].(string)
	if !ok {
		return nil, fmt.Errorf("action must have a string \"action\" field")
	}
	switch action_type {
	case "gather":
		category, err := parseResourceCategory(raw["category"])
		if err != nil {
			return nil, err
		}
		return &gatherAction{category: category}, nil
	case "produce":
		return &produceAction{}, nil
	case "research":
		return &researchAction{}, nil
	case "build":
		id, ok := raw["building_id"].(float64)
		if !ok {
			return nil, fmt.Errorf("build action requires a numeric \"building_id\"")
		}
		action := &buildAction{building_id: uint32(id)}
		if headroom, ok := raw["max_pop_headroom"].(float64); ok {
			action.gate_pop_headroom = true
			action.max_pop_headroom = int(headroom)
		}
		if max_count, ok := raw["max_count"].(float64); ok {
			action.max_count = int(max_count)
		}
		return action, nil
	case "explore":
		return &exploreAction{}, nil
	case "attack":
		action := &attackAction{target: attackTargetMilitary}
		if t, ok := raw["target"].(string); ok {
			switch t {
			case "military":
				action.target = attackTargetMilitary
			case "economic":
				action.target = attackTargetEconomic
			case "any":
				action.target = attackTargetAny
			default:
				return nil, fmt.Errorf("unknown attack target %q", t)
			}
		}
		return action, nil
	default:
		return nil, fmt.Errorf("unknown action type %q", action_type)
	}
}

func parseResourceCategory(raw interface{}) (ResourceCategory, error) {
	s, ok := raw.(string)
	if !ok {
		return 0, fmt.Errorf("gather action requires a string \"category\"")
	}
	switch s {
	case "food":
		return ResourceFood, nil
	case "wood":
		return ResourceWood, nil
	case "stone":
		return ResourceStone, nil
	case "gold":
		return ResourceGold, nil
	default:
		return 0, fmt.Errorf("unknown resource category %q", s)
	}
}
