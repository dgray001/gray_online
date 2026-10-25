package aibridge

import (
	"slices"
	"sort"

	"github.com/dgray001/gray_online/game/games/risq/ai"
	"github.com/dgray001/gray_online/game/games/risq/internal/defs"
	"github.com/dgray001/gray_online/util"
)

// Answers every ai query from the player's frontend snapshot only, so the ai knows exactly what a human would.
type aiView struct {
	std                   *util.Logger
	revision              uint64
	queues                map[uint64][]snapOrder
	queue_subjects        map[uint64][]uint64
	config                *defs.Config
	game                  *snapGame
	me                    *snapPlayer
	spaces                map[ai.Coordinate]*snapSpace
	zones                 []snapZoneEntry
	units                 map[uint64]*snapUnit
	buildings             map[uint64]*snapBuilding
	planned               map[ai.ZoneRef]bool
	order_ids             map[uint64]bool
	claimed_buildings     map[uint64]bool
	ordered_foundations   map[ai.ZoneRef]uint32
	cancelled_foundations map[ai.ZoneRef]bool
	cancelled_orders      map[uint64]bool
	planned_techs         map[uint32]bool
	planned_production    map[uint64]int
	// Cached from the current tasks in the queue mirror
	gather_counts map[ai.ZoneRef]int
	// shared by every caller, so callers must treat it as read-only; dropped when a unit's view inputs change
	units_cache []ai.UnitView
	// the enemy lists, read-only and shared; rebuilt when the revision they were built at has passed
	enemy_cache enemyCache
	// built on first use
	enemy_spaces       []ai.Coordinate
	enemy_spaces_known bool
	space_by_key       map[uint]*snapSpace
	space_links        map[ai.Coordinate][]ai.Coordinate
	linked_edges       map[ai.ZoneRef]ai.ZoneRef
	space_distances    map[ai.Coordinate]map[ai.Coordinate]int
}

type attackTarget struct {
	building bool
	id       uint64
}

type snapZoneEntry struct {
	ref   ai.ZoneRef
	space *snapSpace
	zone  *snapZone
}

func newAiView(config *defs.Config, game *snapGame, player_id int, std *util.Logger) (*aiView, bool) {
	v := &aiView{
		std: std, config: config, game: game, spaces: map[ai.Coordinate]*snapSpace{}, units: map[uint64]*snapUnit{}, buildings: map[uint64]*snapBuilding{},
		planned: map[ai.ZoneRef]bool{}, order_ids: map[uint64]bool{},
		claimed_buildings: map[uint64]bool{}, ordered_foundations: map[ai.ZoneRef]uint32{}, cancelled_foundations: map[ai.ZoneRef]bool{},
		cancelled_orders: map[uint64]bool{}, planned_techs: map[uint32]bool{}, planned_production: map[uint64]int{},
		space_distances: map[ai.Coordinate]map[ai.Coordinate]int{},
	}
	v.indexSpaceLinks()
	for i := range game.Players {
		p := &game.Players[i]
		if p.Player.PlayerId == player_id {
			v.me = p
		}
		for j := range p.Units {
			v.units[p.Units[j].InternalId] = &p.Units[j]
		}
		for j := range p.Buildings {
			v.buildings[p.Buildings[j].InternalId] = &p.Buildings[j]
		}
	}
	if v.me == nil {
		return nil, false
	}
	for _, row := range game.Spaces {
		for _, space := range row {
			if space == nil {
				continue
			}
			v.spaces[toCoordinate(space.Coordinate)] = space
			for _, zone_row := range space.Zones {
				for k := range zone_row {
					ref := ai.ZoneRef{Space: toCoordinate(space.Coordinate), Zone: toCoordinate(zone_row[k].Coordinate)}
					v.zones = append(v.zones, snapZoneEntry{ref: ref, space: space, zone: &zone_row[k]})
				}
			}
		}
	}
	for _, f := range v.me.PlannedFoundations {
		v.planned[zoneRefFromKey(f.CoordinateKey)] = true
	}
	for _, o := range v.me.ActiveOrders {
		v.order_ids[o.InternalId] = true
	}
	return v, true
}

func (v *aiView) zoneAt(ref ai.ZoneRef) *snapZone {
	space := v.spaces[ref.Space]
	if space == nil {
		return nil
	}
	for _, row := range space.Zones {
		for k := range row {
			if toCoordinate(row[k].Coordinate) == ref.Zone {
				return &row[k]
			}
		}
	}
	return nil
}

func (v *aiView) playerId() int {
	return v.me.Player.PlayerId
}

func (v *aiView) Logger() *util.Logger {
	return v.std
}

func (v *aiView) Revision() uint64 {
	return v.revision
}

func (v *aiView) Nickname() string {
	return v.me.Player.Nickname
}

func (v *aiView) resourceNode(r *snapResource, ref ai.ZoneRef) ai.ResourceView {
	return ai.ResourceView{InternalID: r.InternalId, Category: toAiCategory(v.config.Resources[r.ResourceId].Category), Location: ref, AmountLeft: r.ResourcesLeft, GatherSpeed: r.BaseGatherSpeed, GatherCapacity: r.GatherCapacity}
}

func (v *aiView) buildingNode(b *snapBuilding, ref ai.ZoneRef) ai.ResourceView {
	gather := v.config.Buildings[b.BuildingId].Gather
	return ai.ResourceView{InternalID: b.InternalId, Category: toAiCategory(gather.Resource_category), Location: ref, AmountLeft: b.ResourcesLeft, GatherSpeed: gather.Base_gather_speed, GatherCapacity: gather.Gather_capacity, IsBuilding: true}
}

func (v *aiView) gatherTargetAt(ref ai.ZoneRef) (ai.ResourceView, bool) {
	zone := v.zoneAt(ref)
	if zone == nil {
		return ai.ResourceView{}, false
	}
	if r := zone.Resource; r != nil {
		return v.resourceNode(r, ref), true
	}
	if b := zone.Building; b != nil && v.config.Buildings[b.BuildingId].IsGatherable() {
		return v.buildingNode(b, ref), true
	}
	return ai.ResourceView{}, false
}

func (v *aiView) Units() []ai.UnitView {
	if v.units_cache == nil {
		units := make([]ai.UnitView, 0, len(v.me.Units))
		for i := range v.me.Units {
			units = append(units, v.unitView(&v.me.Units[i]))
		}
		v.units_cache = sortUnitViews(units)
	}
	return v.units_cache
}

func (v *aiView) refreshUnit(id uint64) {
	if v.units_cache == nil {
		return
	}
	at := sort.Search(len(v.units_cache), func(i int) bool { return v.units_cache[i].InternalID >= id })
	if at == len(v.units_cache) || v.units_cache[at].InternalID != id {
		return
	}
	// slices already handed out keep their snapshot
	units := slices.Clone(v.units_cache)
	units[at] = v.unitView(v.units[id])
	v.units_cache = units
}

func (v *aiView) IdleUnits() []ai.UnitView {
	return v.EligibleUnits()
}

func (v *aiView) EligibleUnits(kinds ...ai.OrderKind) []ai.UnitView {
	allowed := make(map[ai.OrderKind]bool, len(kinds))
	for _, k := range kinds {
		allowed[k] = true
	}
	units := make([]ai.UnitView, 0)
	for _, u := range v.Units() {
		if u.CurrentOrder == nil || allowed[u.CurrentOrder.Kind] {
			units = append(units, u)
		}
	}
	return units
}

func (v *aiView) Buildings() []ai.BuildingView {
	buildings := make([]ai.BuildingView, 0, len(v.me.Buildings))
	for i := range v.me.Buildings {
		buildings = append(buildings, v.ownBuildingView(&v.me.Buildings[i]))
	}
	return sortBuildingViews(buildings)
}

func (v *aiView) IdleBuildings() []ai.BuildingView {
	buildings := make([]ai.BuildingView, 0)
	for _, b := range v.Buildings() {
		if b.Idle && !v.claimed_buildings[b.InternalID] {
			buildings = append(buildings, b)
		}
	}
	return buildings
}

func (v *aiView) Resource(category ai.ResourceCategory) float64 {
	switch category {
	case ai.ResourceFood:
		return v.me.Resources.Food
	case ai.ResourceWood:
		return v.me.Resources.Wood
	case ai.ResourceStone:
		return v.me.Resources.Stone
	default:
		return v.me.Resources.Gold
	}
}

func (v *aiView) MaxPopulation() int {
	return v.me.MaxPopulationLimit
}

func (v *aiView) HasEconomy() bool {
	return v.game.HasEconomy
}

func (v *aiView) HasProduction() bool {
	return v.game.HasProduction
}

func (v *aiView) Population() (int, int) {
	return len(v.me.Units), v.me.PopulationLimit
}

func (v *aiView) occupied(ref ai.ZoneRef) bool {
	zone := v.zoneAt(ref)
	return zone != nil && (zone.Resource != nil || zone.Building != nil)
}

func (v *aiView) foundationsByLocation() map[ai.ZoneRef]*ai.FoundationView {
	foundations := make(map[ai.ZoneRef]*ai.FoundationView)
	for _, f := range v.me.PlannedFoundations {
		if ref := zoneRefFromKey(f.CoordinateKey); !v.occupied(ref) {
			foundations[ref] = &ai.FoundationView{BuildingID: f.BuildingId, Location: ref, Planned: true}
		}
	}
	for location, building_id := range v.ordered_foundations {
		_, server_known := foundations[location]
		foundations[location] = &ai.FoundationView{BuildingID: building_id, Location: location, Planned: true, OrderedThisTurn: !server_known}
	}
	for _, b := range v.me.Buildings {
		if b.UnderConstruction {
			ref := zoneRefOf(b.Space, b.Zone)
			foundations[ref] = &ai.FoundationView{BuildingID: b.BuildingId, Location: ref}
		}
	}
	for location := range v.cancelled_foundations {
		delete(foundations, location)
	}
	return foundations
}

func (v *aiView) Foundations() []ai.FoundationView {
	by_location := v.foundationsByLocation()
	for _, u := range v.Units() {
		if u.CurrentOrder != nil && u.CurrentOrder.Kind == ai.OrderKindBuild && u.CurrentOrder.TargetZone != nil {
			if f, ok := by_location[*u.CurrentOrder.TargetZone]; ok {
				f.Builders++
			}
		}
	}
	foundations := make([]ai.FoundationView, 0, len(by_location))
	for _, f := range by_location {
		foundations = append(foundations, *f)
	}
	sort.Slice(foundations, func(i, j int) bool { return zoneRefLess(foundations[i].Location, foundations[j].Location) })
	return foundations
}

func (v *aiView) TurnNumber() int {
	return v.game.TurnNumber
}

func (v *aiView) MapSize() int {
	return (len(v.game.Spaces) - 1) / 2
}

func (v *aiView) NumPlayers() int {
	return len(v.game.Players)
}

func (v *aiView) EnemiesFound() int {
	discovered := make(map[int]bool)
	for _, entry := range v.zones {
		if b := entry.zone.Building; b != nil && b.PlayerId != v.playerId() {
			discovered[b.PlayerId] = true
		}
	}
	return len(discovered)
}

func (v *aiView) Score() int {
	return int(v.me.Score)
}

func (v *aiView) BestEnemyScore() int {
	best := 0
	for _, p := range v.game.Players {
		if p.Player.PlayerId != v.playerId() && int(p.Score) > best {
			best = int(p.Score)
		}
	}
	return best
}

func (v *aiView) TechResearched(tech_id uint32) bool {
	return v.me.ResearchedTechs[tech_id]
}

func (v *aiView) techMet(required_tech_id uint32) bool {
	return required_tech_id == 0 || v.me.ResearchedTechs[required_tech_id]
}

func (v *aiView) BuildingAvailable(building_id uint32) bool {
	return v.techMet(v.config.Buildings[building_id].Required_tech_id)
}

func (v *aiView) ValidBuildZone(building_id uint32, target ai.ZoneRef) bool {
	zone := v.zoneAt(target)
	if zone == nil || zone.Resource != nil || v.config.Buildings[building_id].Is_wall && target.Zone == (ai.Coordinate{}) {
		return false
	}
	b := zone.Building
	return b == nil || b.PlayerId == v.playerId() && b.UnderConstruction && b.BuildingId == building_id
}

func (v *aiView) ScopedUnits() []ai.UnitView {
	return v.Units()
}

func (v *aiView) InAttackRange(b ai.BuildingView, target ai.ZoneRef) bool {
	building, ok := v.buildings[b.InternalID]
	if !ok {
		return false
	}
	radius, ranged := building.AttackRange.SpaceRadius()
	if !ranged {
		return b.Location == target
	}
	if radius == 0 && b.Location.Space != target.Space {
		facing, ok := v.facingZone(b.Location)
		return ok && facing == target
	}
	return v.SpaceDistance(b.Location.Space, target.Space) <= int(radius)
}

type enemyCache struct {
	units     []ai.UnitView
	buildings []ai.BuildingView
}

// What we know of the enemy does not depend on the orders we emit, so it is built once per view
func (v *aiView) enemyViews() *enemyCache {
	if v.enemy_cache.units == nil {
		v.enemy_cache = enemyCache{units: v.buildVisibleEnemyUnits(), buildings: v.buildKnownEnemyBuildings()}
	}
	return &v.enemy_cache
}

func (v *aiView) VisibleEnemyUnits() []ai.UnitView {
	return v.enemyViews().units
}

func (v *aiView) buildVisibleEnemyUnits() []ai.UnitView {
	units := make([]ai.UnitView, 0)
	for i := range v.game.Players {
		p := &v.game.Players[i]
		if p == v.me {
			continue
		}
		for j := range p.Units {
			view := v.unitViewShallow(&p.Units[j])
			view.CurrentOrder = v.currentOrder(p.Units[j].ActiveOrders)
			units = append(units, view)
		}
	}
	return sortUnitViews(units)
}

func (v *aiView) AllUnits() []ai.UnitView {
	return slices.Concat(v.Units(), v.VisibleEnemyUnits())
}

func (v *aiView) AllBuildings() []ai.BuildingView {
	return slices.Concat(v.Buildings(), v.KnownEnemyBuildings())
}

func (v *aiView) VisibleEnemyBuildings() []ai.BuildingView {
	buildings := make([]ai.BuildingView, 0)
	for i := range v.game.Players {
		p := &v.game.Players[i]
		if p == v.me {
			continue
		}
		for j := range p.Buildings {
			buildings = append(buildings, v.buildingView(&p.Buildings[j]))
		}
	}
	return sortBuildingViews(buildings)
}

// Visible enemy buildings plus the ones last seen in spaces now under fog (they may since have been destroyed)
func (v *aiView) KnownEnemyBuildings() []ai.BuildingView {
	return v.enemyViews().buildings
}

func (v *aiView) buildKnownEnemyBuildings() []ai.BuildingView {
	seen := make(map[uint64]bool)
	buildings := make([]ai.BuildingView, 0)
	for _, b := range v.VisibleEnemyBuildings() {
		seen[b.InternalID] = true
		buildings = append(buildings, b)
	}
	for _, space := range v.spaces {
		for i := range space.Buildings {
			b := &space.Buildings[i]
			if b.PlayerId == v.playerId() || seen[b.InternalId] {
				continue
			}
			seen[b.InternalId] = true
			remembered := v.buildingView(b)
			remembered.Visible = false
			buildings = append(buildings, remembered)
		}
	}
	return sortBuildingViews(buildings)
}

func (v *aiView) gatherableAt(entry snapZoneEntry, category ai.ResourceCategory) (ai.ResourceView, bool) {
	if r := entry.zone.Resource; r != nil {
		view := v.resourceNode(r, entry.ref)
		return view, r.ResourcesLeft > 0 && view.Category == category
	}
	b := entry.zone.Building
	if b == nil || b.PlayerId != v.playerId() || b.UnderConstruction || b.ResourcesLeft <= 0 {
		return ai.ResourceView{}, false
	}
	config := v.config.Buildings[b.BuildingId]
	if !config.IsGatherable() || toAiCategory(config.Gather.Resource_category) != category {
		return ai.ResourceView{}, false
	}
	return v.buildingNode(b, entry.ref), true
}

func (v *aiView) gathererCount(location ai.ZoneRef) int {
	if v.gather_counts == nil {
		v.gather_counts = make(map[ai.ZoneRef]int)
		for _, u := range v.Units() {
			if u.CurrentOrder != nil && u.CurrentOrder.TargetResource != nil {
				v.gather_counts[u.CurrentOrder.TargetResource.Location]++
			}
		}
	}
	return v.gather_counts[location]
}

func (v *aiView) gatherTarget(unit_id uint64) (ai.ZoneRef, bool) {
	v.initQueues()
	order := v.currentOrder(v.queues[unit_id])
	if order == nil || order.TargetResource == nil {
		return ai.ZoneRef{}, false
	}
	return order.TargetResource.Location, true
}

// Prefers the nearest node its current gatherers won't already empty, so nearly empty nodes still get finished.
func (v *aiView) NearestResource(from ai.ZoneRef, category ai.ResourceCategory, gatherer_id uint64) (ai.ResourceView, bool) {
	own_target, has_own_target := v.gatherTarget(gatherer_id)
	var best ai.ResourceView
	found, best_distance, best_saturated := false, -1, false
	for _, entry := range v.zones {
		view, ok := v.gatherableAt(entry, category)
		if !ok {
			continue
		}
		count := v.gathererCount(entry.ref)
		if has_own_target && own_target == entry.ref {
			count--
		}
		if count >= view.GatherCapacity {
			continue
		}
		// saturated only when the gatherers already on it will use up what is left, so a nearly empty node still gets finished
		saturated := view.AmountLeft <= float64(count*view.GatherSpeed)
		d := v.LocationDistance(from, entry.ref)
		closer := d < best_distance || (d == best_distance && zoneKey(entry.ref) < zoneKey(best.Location))
		if !found || (!saturated && best_saturated) || (saturated == best_saturated && closer) {
			best, best_distance, best_saturated, found = view, d, saturated, true
		}
	}
	return best, found
}

func (v *aiView) KnownResources(category ai.ResourceCategory) []ai.ResourceView {
	resources := make([]ai.ResourceView, 0)
	for _, entry := range v.zones {
		if r := entry.zone.Resource; r != nil && r.ResourcesLeft > 0 && toAiCategory(v.config.Resources[r.ResourceId].Category) == category {
			resources = append(resources, v.resourceNode(r, entry.ref))
		}
	}
	return resources
}

func (v *aiView) BuildCost(building_id uint32) ai.Cost {
	config, ok := v.config.Buildings[building_id]
	if !ok || config.Build_stamina <= 0 {
		return ai.Cost{}
	}
	return toCost(config.Cost)
}

func (v *aiView) UnitCost(unit_id uint32) ai.Cost {
	config, ok := v.config.Units[unit_id]
	if !ok {
		return ai.Cost{}
	}
	return toCost(config.Cost)
}

func (v *aiView) TechCost(tech_id uint32) ai.Cost {
	config, ok := v.config.Techs[tech_id]
	if !ok {
		return ai.Cost{}
	}
	return toCost(config.Cost)
}

func (v *aiView) ConfigValue(table string, field string, id uint32) (float64, error) {
	return v.config.ConfigValue(table, field, id)
}

func (v *aiView) claimBuilding(id uint64) {
	v.claimed_buildings[id] = true
	v.revision++
}

func (v *aiView) PlayerIDs() []int {
	ids := make([]int, 0, len(v.game.Players))
	for _, p := range v.game.Players {
		ids = append(ids, p.Player.PlayerId)
	}
	sort.Ints(ids)
	return ids
}

func (v *aiView) PlayerID() int {
	return v.playerId()
}

func (v *aiView) PlayerScore(player_id int) int {
	for _, p := range v.game.Players {
		if p.Player.PlayerId == player_id {
			return int(p.Score)
		}
	}
	return 0
}

func (v *aiView) PlayerEliminated(player_id int) bool {
	for _, p := range v.game.Players {
		if p.Player.PlayerId == player_id {
			return p.Eliminated
		}
	}
	return false
}
