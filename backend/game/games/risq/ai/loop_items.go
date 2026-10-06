package ai

import "slices"

// One thing a for_each is visiting; its numeric fields are read in expressions as var(name.field)
type loopItem interface {
	field(name string) (float64, bool)
}

// Where a located item is, and how many space steps that is from home
type place struct {
	loc           ZoneRef
	distance_home int
}

var placeFields = []string{"x", "y", "zone_x", "zone_y", "distance_home"}

func (p place) field(name string) (float64, bool) {
	switch name {
	case "x":
		return float64(p.loc.Space.X), true
	case "y":
		return float64(p.loc.Space.Y), true
	case "zone_x":
		return float64(p.loc.Zone.X), true
	case "zone_y":
		return float64(p.loc.Zone.Y), true
	case "distance_home":
		return float64(p.distance_home), true
	}
	return 0, false
}

// Home is looked up once per loop, since it scans every building
type homePoint struct {
	loc   ZoneRef
	known bool
}

func newHomePoint(view View) homePoint {
	loc, known := homeLocation(view)
	return homePoint{loc: loc, known: known}
}

func (h homePoint) placeOf(view View, loc ZoneRef) place {
	p := place{loc: loc}
	if h.known {
		p.distance_home = view.SpaceDistance(h.loc.Space, loc.Space)
	}
	return p
}

type playerItem struct {
	id, score, units_visible, buildings_known int
	me                                        bool
}

func (p playerItem) field(name string) (float64, bool) {
	switch name {
	case "id":
		return float64(p.id), true
	case "is_me":
		return boolNumber(p.me), true
	case "score":
		return float64(p.score), true
	case "units_visible":
		return float64(p.units_visible), true
	case "buildings_known":
		return float64(p.buildings_known), true
	}
	return 0, false
}

// unit_count is -1 when the space's units are not known; unidentified_count is the units there that are counted but not seen
type spaceItem struct {
	place
	vision, owner, unit_count, unidentified_count int
}

func (s spaceItem) field(name string) (float64, bool) {
	switch name {
	case "vision":
		return float64(s.vision), true
	case "owner":
		return float64(s.owner), true
	case "unit_count":
		return float64(s.unit_count), true
	case "unidentified_count":
		return float64(s.unidentified_count), true
	}
	return s.place.field(name)
}

type unitItem struct {
	place
	unit UnitView
}

func (u unitItem) field(name string) (float64, bool) {
	switch name {
	case "id":
		return float64(u.unit.InternalID), true
	case "player":
		return float64(u.unit.PlayerID), true
	case "unit_id":
		return float64(u.unit.UnitID), true
	case "unit_type":
		return float64(u.unit.Type), true
	case "is_military":
		return boolNumber(u.unit.Kind == UnitMilitary), true
	case "is_garrisoned":
		return boolNumber(u.unit.GarrisonedIn != nil), true
	case "health":
		return u.unit.Health, true
	case "max_health":
		return u.unit.MaxHealth, true
	case "stamina":
		return float64(u.unit.CurrentStamina), true
	}
	return u.place.field(name)
}

type buildingItem struct {
	place
	building BuildingView
}

func (b buildingItem) field(name string) (float64, bool) {
	switch name {
	case "id":
		return float64(b.building.InternalID), true
	case "player":
		return float64(b.building.PlayerID), true
	case "building_id":
		return float64(b.building.BuildingID), true
	case "is_under_construction":
		return boolNumber(b.building.UnderConstruction), true
	case "health":
		return b.building.Health, true
	case "max_health":
		return b.building.MaxHealth, true
	}
	return b.place.field(name)
}

type resourceItem struct {
	place
	resource ResourceView
}

func (r resourceItem) field(name string) (float64, bool) {
	switch name {
	case "id":
		return float64(r.resource.InternalID), true
	case "category":
		return float64(r.resource.Category), true
	case "amount_left":
		return r.resource.AmountLeft, true
	}
	return r.place.field(name)
}

// building_player is -1 with no building
type zoneItem struct {
	place
	has_resource    bool
	building_player int
}

func (z zoneItem) field(name string) (float64, bool) {
	switch name {
	case "has_resource":
		return boolNumber(z.has_resource), true
	case "building_player":
		return float64(z.building_player), true
	}
	return z.place.field(name)
}

func withPlaceFields(names ...string) []string {
	return append(names, placeFields...)
}

// The fields each source's items have; checked when a config loads, and kept in step with the item types by a test
var itemFields = map[string][]string{
	"players":   {"id", "is_me", "score", "units_visible", "buildings_known"},
	"spaces":    withPlaceFields("vision", "owner", "unit_count", "unidentified_count"),
	"units":     withPlaceFields("id", "player", "unit_id", "unit_type", "is_military", "is_garrisoned", "health", "max_health", "stamina"),
	"buildings": withPlaceFields("id", "player", "building_id", "is_under_construction", "health", "max_health"),
	"resources": withPlaceFields("id", "category", "amount_left"),
	"zones":     withPlaceFields("has_resource", "building_player"),
}

func hasItemField(source string, name string) bool {
	return slices.Contains(itemFields[source], name)
}
