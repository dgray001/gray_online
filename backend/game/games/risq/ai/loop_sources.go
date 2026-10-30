package ai

import (
	"fmt"
	"iter"
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
		"buildings": parseBuildingSource, "resources": parseResourceSource, "zones": parseZoneSource, "regions": parseRegionSource,
		"mercenaries": parseMercenarySource, "foundations": parseFoundationSource,
		"gatherables": parseGatherableSource, "attackables": parseAttackableSource,
		"orders": parseOrderSource, "assignments": parseAssignmentSource,
	}
}

type mercenarySource struct{}

func parseMercenarySource(where map[string]any) (loopSource, error) {
	if err := checkKeys(where); err != nil {
		return nil, err
	}
	return &mercenarySource{}, nil
}

func (s *mercenarySource) items(view View, _ *Internals) []loopItem {
	items := make([]loopItem, 0, len(view.AvailableMercenaries()))
	for _, mercenary := range view.AvailableMercenaries() {
		items = append(items, mercenaryItem{producible: mercenary})
	}
	return items
}

type regionSource struct{}

func parseRegionSource(where map[string]any) (loopSource, error) {
	if err := checkKeys(where); err != nil {
		return nil, err
	}
	return &regionSource{}, nil
}

func (s *regionSource) items(view View, internals *Internals) []loopItem {
	stats := internals.worldFor(view).regions.stats
	items := make([]loopItem, 0, len(stats))
	for _, stat := range stats {
		items = append(items, regionItem{stat: stat})
	}
	return items
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
		if s.selector.allows(view, internals, id) {
			items = append(items, playerItem{id: id, me: id == view.PlayerID(), eliminated: view.PlayerEliminated(id), score: view.PlayerScore(id), units_visible: units[id], buildings_known: buildings[id]})
		}
	}
	return items
}

// The existing space condition, with expression-valued player ids added
type spaceSource struct {
	condition               SpaceCondition
	player, building_player *amount
	matchesEverySpace       bool
}

type unitSource struct {
	selector selector
	filter   unitFilter
}

func parseUnitSource(where map[string]any) (loopSource, error) {
	if err := checkKeys(where, "owner", "player_id", "unit_ids", "unit_types"); err != nil {
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
	if r == relationUnowned {
		return nil
	}
	if r == relationAny {
		return view.AllUnits()
	}
	if r == relationOwn {
		return view.Units()
	}
	return view.VisibleEnemyUnits()
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
	if s.player, err = takeExpression(rest, "player_id"); err != nil {
		return nil, err
	}
	if s.building_player, err = takeExpression(rest, "building_player_id"); err != nil {
		return nil, err
	}
	s.matchesEverySpace = len(rest) == 0 && s.player == nil && s.building_player == nil
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
	coordinates := view.MatchingSpaces(condition)
	slices.SortFunc(coordinates, compareCoordinates)
	return s.itemsAt(coordinates, view, internals)
}

func (s *spaceSource) itemsAt(coordinates []Coordinate, view View, internals *Internals) []loopItem {
	infos, own := internals.worldFor(view).spaces, ownUnitsBySpace(view)
	items := make([]loopItem, 0, len(coordinates))
	for _, coordinate := range coordinates {
		items = append(items, newSpaceItem(coordinate, infos[coordinate], own))
	}
	return items
}

func newSpaceItem(coordinate Coordinate, info SpaceInfo, own map[Coordinate]int) spaceItem {
	item := spaceItem{place: place{loc: ZoneRef{Space: coordinate}}, info: info, vision: int(info.Vision), owner: info.Owner, unit_count: -1, unidentified_count: unidentifiedUnitCount(info, own)}
	if info.UnitCount != nil {
		item.unit_count = *info.UnitCount
	}
	return item
}

type buildingSource struct {
	selector selector
	ids      buildingFilter
	state    func(BuildingView) bool
}

func parseBuildingSource(where map[string]any) (loopSource, error) {
	if err := checkKeys(where, "owner", "player_id", "building_ids", "state"); err != nil {
		return nil, err
	}
	if err := checkOptionTypes[string](where, "state"); err != nil {
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
func buildingsFor(view View, r relation) iter.Seq[BuildingView] {
	return func(yield func(BuildingView) bool) {
		if r == relationUnowned {
			return
		}
		mine := r == relationOwn
		for _, b := range view.AllBuildings() {
			if (r == relationAny || (b.PlayerID == view.PlayerID()) == mine) && !yield(b) {
				return
			}
		}
	}
}

func (s *buildingSource) items(view View, internals *Internals) []loopItem {
	items := make([]loopItem, 0)
	for b := range buildingsFor(view, s.selector.relation) {
		if s.ids.matches(b.BuildingID) && s.state(b) && s.selector.allows(view, internals, b.PlayerID) {
			items = append(items, buildingItem{place: place{loc: b.Location}, building: b})
		}
	}
	return items
}

type resourceSource struct {
	categories []ResourceCategory
}

func parseResourceSource(where map[string]any) (loopSource, error) {
	source, err := newResourceSource(where)
	return source, err
}

func newResourceSource(where map[string]any) (*resourceSource, error) {
	if err := checkKeys(where, "category"); err != nil {
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
	return s, nil
}

func (s *resourceSource) takes(category ResourceCategory) bool {
	return slices.Contains(s.categories, category)
}

func (s *resourceSource) nodes(view View) []ResourceView {
	nodes := make([]ResourceView, 0)
	for _, category := range s.categories {
		nodes = append(nodes, view.KnownResources(category)...)
	}
	return nodes
}

func (s *resourceSource) items(view View, internals *Internals) []loopItem {
	items := make([]loopItem, 0)
	for _, r := range s.nodes(view) {
		items = append(items, resourceItem{place: place{loc: r.Location}, resource: r})
	}
	return items
}

// A zone's owner is the player whose building stands on it, so "unowned" means no building; "resource" and "building" filter on what it holds
type zoneSource struct {
	selector           selector
	resource, building *bool
}

func parseZoneSource(where map[string]any) (loopSource, error) {
	if err := checkKeys(where, "owner", "player_id", "resource", "building"); err != nil {
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

func (s *zoneSource) item(view View, internals *Internals, z ZoneInfo) (loopItem, bool) {
	has_building := z.BuildingPlayer >= 0
	if (s.resource != nil && *s.resource != z.HasResource) || (s.building != nil && *s.building != has_building) {
		return nil, false
	}
	if !s.selector.allows(view, internals, z.BuildingPlayer) {
		return nil, false
	}
	return newZoneItem(z), true
}

func newZoneItem(z ZoneInfo) zoneItem {
	return zoneItem{place: place{loc: z.Location}, has_resource: z.HasResource, building_player: z.BuildingPlayer}
}

func (s *zoneSource) items(view View, internals *Internals) []loopItem {
	items := make([]loopItem, 0)
	world := internals.worldFor(view)
	world.loadZones(view)
	for _, z := range world.zones {
		if item, ok := s.item(view, internals, z); ok {
			items = append(items, item)
		}
	}
	return items
}

func (s *unitSource) item(view View, internals *Internals, u UnitView) (loopItem, bool) {
	if s.filter.matches(u) && s.selector.allows(view, internals, u.PlayerID) {
		return unitItem{place: place{loc: u.Location}, unit: u}, true
	}
	return nil, false
}

func (s *unitSource) items(view View, internals *Internals) []loopItem {
	items := make([]loopItem, 0)
	for _, u := range unitsFor(view, s.selector.relation) {
		if item, ok := s.item(view, internals, u); ok {
			items = append(items, item)
		}
	}
	return items
}
