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

func (p place) field(name string) (float64, bool) {
	return placeFields.get(p, name)
}

type playerItem struct {
	id, score, units_visible, buildings_known int
	me                                        bool
}

func (p playerItem) field(name string) (float64, bool) {
	return playerFields.get(p, name)
}

// unit_count is -1 when the space's units are not known; unidentified_count is the units there that are counted but not seen
type spaceItem struct {
	place
	info                                          SpaceInfo
	vision, owner, unit_count, unidentified_count int
}

func (s spaceItem) field(name string) (float64, bool) {
	return spaceFields.get(s, name)
}

type regionItem struct{ stat regionStat }

func (r regionItem) field(name string) (float64, bool) {
	return regionFields.get(r, name)
}

type mercenaryItem struct{ producible Producible }

func (m mercenaryItem) field(name string) (float64, bool) {
	return mercenaryFields.get(m, name)
}

type unitItem struct {
	place
	unit UnitView
}

func (u unitItem) field(name string) (float64, bool) {
	return unitFields.get(u, name)
}

type buildingItem struct {
	place
	building BuildingView
}

func (b buildingItem) field(name string) (float64, bool) {
	return buildingFields.get(b, name)
}

type resourceItem struct {
	place
	resource ResourceView
}

func (r resourceItem) field(name string) (float64, bool) {
	return resourceFields.get(r, name)
}

// building_player is -1 with no building
type zoneItem struct {
	place
	has_resource    bool
	building_player int
}

func (z zoneItem) field(name string) (float64, bool) {
	return zoneFields.get(z, name)
}

// The fields each source's items have; checked when a config loads
var itemFields = map[string][]string{
	"players":     playerFields.names(),
	"spaces":      spaceFields.names(),
	"regions":     regionFields.names(),
	"mercenaries": mercenaryFields.names(),
	"units":       unitFields.names(),
	"buildings":   buildingFields.names(),
	"resources":   resourceFields.names(),
	"zones":       zoneFields.names(),
	"foundations": foundationFields.names(),
	"gatherables": gatherableFields.names(),
	"attackables": attackableFields.names(),
}

func hasItemField(source string, name string) bool {
	return slices.Contains(itemFields[source], name)
}
