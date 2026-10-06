package ai

import (
	"fmt"
	"maps"
	"slices"
)

// What a for_each can visit; items are returned in a fixed order, so a loop never depends on map order
type loopSource interface {
	items(view View, internals *Internals) []loopItem
}

var loopSourceParsers map[string]func(where map[string]any) (loopSource, error)

// Filled in init: the parsers reach parseAmount, which reads counterParsers
func init() {
	loopSourceParsers = map[string]func(where map[string]any) (loopSource, error){
		"players": parsePlayerSource, "spaces": parseSpaceSource, "units": parseUnitSource,
		"buildings": parseBuildingSource, "resources": parseResourceSource, "zones": parseZoneSource,
	}
}

type playerSource struct {
	selector selector
}

func parsePlayerSource(where map[string]any) (loopSource, error) {
	if err := checkKeys(where, "owner", "player_id"); err != nil {
		return nil, err
	}
	s, err := parseSelector(where, relationAny)
	return &playerSource{selector: s}, err
}

func (s *playerSource) items(view View, internals *Internals) []loopItem {
	units, buildings := map[int]int{}, map[int]int{}
	for _, u := range slices.Concat(view.Units(), view.VisibleEnemyUnits()) {
		units[u.PlayerID]++
	}
	for _, b := range slices.Concat(view.Buildings(), view.KnownEnemyBuildings()) {
		buildings[b.PlayerID]++
	}
	items := make([]loopItem, 0)
	for _, id := range view.PlayerIDs() {
		if s.selector.allows(view, internals, id, ZoneRef{}) {
			items = append(items, playerItem{id: id, me: id == view.PlayerID(), score: view.PlayerScore(id), units_visible: units[id], buildings_known: buildings[id]})
		}
	}
	return items
}

// The existing space condition (so a loop and a count_spaces take the same keys), with "within"/"from" and expression-valued player ids added
type spaceSource struct {
	condition               SpaceCondition
	near                    nearFilter
	player, building_player *amount
}

type unitSource struct {
	selector selector
	filter   unitFilter
}

func parseUnitSource(where map[string]any) (loopSource, error) {
	if err := checkKeys(where, "owner", "player_id", "within", "from", "unit_ids", "unit_types"); err != nil {
		return nil, err
	}
	s, err := parseSelector(where, relationOwn)
	if err != nil {
		return nil, err
	}
	filter, err := parseUnitFilter(where)
	return &unitSource{selector: s, filter: filter}, err
}

// The units an owner filter can match: ours, the visible enemy's, or both
func unitsFor(view View, r relation) []UnitView {
	switch r {
	case relationOwn:
		return view.Units()
	case relationEnemy:
		return view.VisibleEnemyUnits()
	case relationUnowned:
		return nil
	}
	return slices.Concat(view.Units(), view.VisibleEnemyUnits())
}

func compareCoordinates(a Coordinate, b Coordinate) int {
	if coordinateLess(a, b) {
		return -1
	}
	if coordinateLess(b, a) {
		return 1
	}
	return 0
}

// Takes an expression-valued key out of where, leaving plain numbers for the space condition
func takeExpression(where map[string]any, key string) (*amount, error) {
	raw, isString := where[key].(string)
	if !isString {
		return nil, nil
	}
	delete(where, key)
	a, err := parseAmount(raw)
	return &a, err
}

func parseSpaceSource(where map[string]any) (loopSource, error) {
	s := &spaceSource{}
	rest := maps.Clone(where)
	var err error
	if s.near, err = parseNearFilter(where); err != nil {
		return nil, err
	}
	delete(rest, "within")
	delete(rest, "from")
	if s.player, err = takeExpression(rest, "player_id"); err != nil {
		return nil, err
	}
	if s.building_player, err = takeExpression(rest, "building_player_id"); err != nil {
		return nil, err
	}
	s.condition, err = parseSpaceCondition(rest)
	return s, err
}

func (s *spaceSource) items(view View, internals *Internals) []loopItem {
	condition := s.condition
	if s.player != nil {
		id := s.player.int(view, internals)
		condition.PlayerID = &id
	}
	if s.building_player != nil {
		id := s.building_player.int(view, internals)
		condition.BuildingPlayerID = &id
	}
	infos := make(map[Coordinate]SpaceInfo)
	for _, info := range view.AllSpaces() {
		infos[info.Space] = info
	}
	coordinates := view.MatchingSpaces(condition)
	slices.SortFunc(coordinates, compareCoordinates)
	home := newHomePoint(view)
	items := make([]loopItem, 0, len(coordinates))
	for _, coordinate := range coordinates {
		loc := ZoneRef{Space: coordinate}
		if !s.near.contains(view, internals, loc) {
			continue
		}
		info := infos[coordinate]
		item := spaceItem{place: home.placeOf(view, loc), vision: int(info.Vision), owner: info.Owner, unit_count: -1, unidentified_count: unidentifiedUnitCount(view, info)}
		if info.UnitCount != nil {
			item.unit_count = *info.UnitCount
		}
		items = append(items, item)
	}
	return items
}

type buildingSource struct {
	selector selector
	ids      buildingFilter
	state    func(BuildingView) bool
}

func parseBuildingSource(where map[string]any) (loopSource, error) {
	if err := checkKeys(where, "owner", "player_id", "within", "from", "building_ids", "state"); err != nil {
		return nil, err
	}
	s, err := parseSelector(where, relationOwn)
	if err != nil {
		return nil, err
	}
	ids, err := parseIDSet(where, "building_ids")
	source := &buildingSource{selector: s, ids: buildingFilter{ids: ids}, state: func(BuildingView) bool { return true }}
	if name, ok := where["state"].(string); ok && err == nil {
		if source.state, ok = buildingStates[name]; !ok {
			err = fmt.Errorf("unknown building state %q", name)
		}
	}
	return source, err
}

// The buildings an owner filter can match: ours, the known enemy's, or both
func buildingsFor(view View, r relation) []BuildingView {
	switch r {
	case relationOwn:
		return view.Buildings()
	case relationEnemy:
		return view.KnownEnemyBuildings()
	case relationUnowned:
		return nil
	}
	return slices.Concat(view.Buildings(), view.KnownEnemyBuildings())
}

func (s *buildingSource) items(view View, internals *Internals) []loopItem {
	home := newHomePoint(view)
	items := make([]loopItem, 0)
	for _, b := range buildingsFor(view, s.selector.relation) {
		if s.ids.matches(b.BuildingID) && s.state(b) && s.selector.allows(view, internals, b.PlayerID, b.Location) {
			items = append(items, buildingItem{place: home.placeOf(view, b.Location), building: b})
		}
	}
	return items
}

type resourceSource struct {
	categories []ResourceCategory
	near       nearFilter
}

func parseResourceSource(where map[string]any) (loopSource, error) {
	if err := checkKeys(where, "category", "within", "from"); err != nil {
		return nil, err
	}
	s := &resourceSource{categories: []ResourceCategory{ResourceFood, ResourceWood, ResourceStone, ResourceGold}}
	if raw, present := where["category"]; present {
		category, err := parseResourceCategory(raw)
		if err != nil {
			return nil, err
		}
		s.categories = []ResourceCategory{category}
	}
	var err error
	s.near, err = parseNearFilter(where)
	return s, err
}

func (s *resourceSource) items(view View, internals *Internals) []loopItem {
	home := newHomePoint(view)
	items := make([]loopItem, 0)
	for _, category := range s.categories {
		for _, r := range view.KnownResources(category) {
			if s.near.contains(view, internals, r.Location) {
				items = append(items, resourceItem{place: home.placeOf(view, r.Location), resource: r})
			}
		}
	}
	return items
}

// A zone's owner is the player whose building stands on it, so "unowned" means no building; "resource" and "building" filter on what it holds
type zoneSource struct {
	selector           selector
	resource, building *bool
}

func parseZoneSource(where map[string]any) (loopSource, error) {
	if err := checkKeys(where, "owner", "player_id", "within", "from", "resource", "building"); err != nil {
		return nil, err
	}
	s, err := parseSelector(where, relationAny)
	if err != nil {
		return nil, err
	}
	source := &zoneSource{selector: s}
	for key, target := range map[string]**bool{"resource": &source.resource, "building": &source.building} {
		if raw, present := where[key]; present {
			flag, ok := raw.(bool)
			if !ok {
				return nil, fmt.Errorf("%q must be a bool", key)
			}
			*target = &flag
		}
	}
	return source, nil
}

func (s *zoneSource) items(view View, internals *Internals) []loopItem {
	home := newHomePoint(view)
	items := make([]loopItem, 0)
	for _, z := range view.AllZones() {
		has_building := z.BuildingPlayer >= 0
		if (s.resource != nil && *s.resource != z.HasResource) || (s.building != nil && *s.building != has_building) {
			continue
		}
		if s.selector.allows(view, internals, z.BuildingPlayer, z.Location) {
			items = append(items, zoneItem{place: home.placeOf(view, z.Location), has_resource: z.HasResource, building_player: z.BuildingPlayer})
		}
	}
	return items
}

func (s *unitSource) items(view View, internals *Internals) []loopItem {
	home := newHomePoint(view)
	items := make([]loopItem, 0)
	for _, u := range unitsFor(view, s.selector.relation) {
		if s.filter.matches(u) && s.selector.allows(view, internals, u.PlayerID, u.Location) {
			items = append(items, unitItem{place: home.placeOf(view, u.Location), unit: u})
		}
	}
	return items
}
