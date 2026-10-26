package ai

import "fmt"

type attackableItem struct {
	place
	unit               *UnitView
	building           *BuildingView
	uid                uint64
	owner, type_id     int
	health, max_health float64
	is_military        bool
	is_visible         bool
}

func (a attackableItem) field(name string) (float64, bool) {
	return attackableFields.get(a, name)
}

func unitAttackable(u UnitView) attackableItem {
	return attackableItem{
		place: place{loc: u.Location}, unit: &u, uid: unitUID(u.InternalID), owner: u.PlayerID, type_id: int(u.UnitID),
		health: u.Health, max_health: u.MaxHealth, is_military: u.Kind == UnitMilitary, is_visible: true,
	}
}

func buildingAttackable(b BuildingView) attackableItem {
	return attackableItem{
		place: place{loc: b.Location}, building: &b, uid: buildingUID(b.InternalID), owner: b.PlayerID, type_id: int(b.BuildingID),
		health: b.Health, max_health: b.MaxHealth, is_visible: b.Visible,
	}
}

var attackableFields = withPlace(fieldTable[attackableItem]{
	"uid":             func(a attackableItem) float64 { return float64(a.uid) },
	"owner":           func(a attackableItem) float64 { return float64(a.owner) },
	"type_id":         func(a attackableItem) float64 { return float64(a.type_id) },
	"health":          func(a attackableItem) float64 { return a.health },
	"max_health":      func(a attackableItem) float64 { return a.max_health },
	"is_building":     func(a attackableItem) float64 { return boolNumber(a.building != nil) },
	"target_category": func(a attackableItem) float64 { return float64(a.targetCategory()) },
	"is_military":     func(a attackableItem) float64 { return boolNumber(a.is_military) },
	"is_visible":      func(a attackableItem) float64 { return boolNumber(a.is_visible) },
}, func(a attackableItem) place { return a.place })

func (a attackableItem) targetCategory() TargetCategory {
	switch {
	case a.building != nil:
		return TargetCategoryBuilding
	case a.is_military:
		return TargetCategoryMilitary
	}
	return TargetCategoryEconomic
}

func attackableAim(a attackableItem) aim {
	if a.building != nil {
		return aim{building: a.building, loc: a.loc}
	}
	return aim{unit: a.unit, loc: a.loc}
}

type attackableSource struct{ selector selector }

func parseAttackableSource(where map[string]any) (loopSource, error) {
	if err := checkKeys(where, "owner", "player_id"); err != nil {
		return nil, err
	}
	s, err := parseSelector(where, relationEnemy)
	if err == nil && s.relation == relationOwn {
		err = fmt.Errorf("attackables never include your own units or buildings")
	}
	return &attackableSource{selector: s}, err
}

func (s *attackableSource) items(view View, internals *Internals) []loopItem {
	items := make([]loopItem, 0)
	for _, u := range unitsFor(view, s.selector.relation) {
		if u.PlayerID != view.PlayerID() && s.selector.allows(view, internals, u.PlayerID) {
			items = append(items, unitAttackable(u))
		}
	}
	for _, b := range buildingsFor(view, s.selector.relation) {
		if b.PlayerID != view.PlayerID() && s.selector.allows(view, internals, b.PlayerID) {
			items = append(items, buildingAttackable(b))
		}
	}
	return items
}
