package aibridge

import (
	"sort"

	"github.com/dgray001/gray_online/game/game_utils"
	"github.com/dgray001/gray_online/game/games/risq/ai"
	"github.com/dgray001/gray_online/game/games/risq/internal/defs"
	"github.com/dgray001/gray_online/util"
)

// Answers every ai query from the player's frontend snapshot only, so the ai knows exactly what a human would.
type aiView struct {
	game                  *snapGame
	me                    *snapPlayer
	spaces                map[ai.Coordinate]*snapSpace
	zones                 []snapZoneEntry
	units                 map[uint64]*snapUnit
	buildings             map[uint64]*snapBuilding
	planned               map[ai.ZoneRef]bool
	order_ids             map[uint64]bool
	assigned_units        map[uint64]*ai.CurrentOrder
	claimed_buildings     map[uint64]bool
	ordered_foundations   map[ai.ZoneRef]uint32
	cancelled_foundations map[ai.ZoneRef]bool
	// built on first use, then kept current by assignUnit
	gather_counts map[ai.ZoneRef]int
}

type snapZoneEntry struct {
	ref   ai.ZoneRef
	space *snapSpace
	zone  *snapZone
}

func newAiView(game *snapGame, player_id int) (*aiView, bool) {
	v := &aiView{
		game: game, spaces: map[ai.Coordinate]*snapSpace{}, units: map[uint64]*snapUnit{}, buildings: map[uint64]*snapBuilding{},
		planned: map[ai.ZoneRef]bool{}, order_ids: map[uint64]bool{}, assigned_units: map[uint64]*ai.CurrentOrder{},
		claimed_buildings: map[uint64]bool{}, ordered_foundations: map[ai.ZoneRef]uint32{}, cancelled_foundations: map[ai.ZoneRef]bool{},
	}
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

func (v *aiView) Nickname() string {
	return v.me.Player.Nickname
}

func (v *aiView) gatherTargetAt(ref ai.ZoneRef) (ai.ResourceView, bool) {
	zone := v.zoneAt(ref)
	if zone == nil {
		return ai.ResourceView{}, false
	}
	if r := zone.Resource; r != nil {
		return ai.ResourceView{InternalID: r.InternalId, Category: toAiCategory(defs.ResourceConfigs[r.ResourceId].Category), Location: ref, AmountLeft: r.ResourcesLeft}, true
	}
	if b := zone.Building; b != nil && defs.BuildingConfigs[b.BuildingId].IsGatherable() {
		return ai.ResourceView{InternalID: b.InternalId, Category: toAiCategory(defs.BuildingConfigs[b.BuildingId].Gather.Resource_category), Location: ref, AmountLeft: b.ResourcesLeft}, true
	}
	return ai.ResourceView{}, false
}

func (v *aiView) currentOrder(orders []snapOrder) *ai.CurrentOrder {
	if len(orders) == 0 {
		return nil
	}
	active := orders[0]
	kind, ok := toOrderKind(active.OrderType)
	if !ok {
		return nil
	}
	order := &ai.CurrentOrder{Kind: kind}
	switch active.OrderType {
	case defs.OrderType_UnitMoveZone, defs.OrderType_UnitAttackZone:
		ref := zoneRefFromKey(uint(active.TargetId))
		order.TargetZone = &ref
	case defs.OrderType_UnitGather:
		if resource, ok := v.gatherTargetAt(zoneRefFromKey(uint(active.TargetId))); ok {
			order.TargetResource = &resource
		}
	case defs.OrderType_UnitBuild:
		_, zone_key := util.InvertPair(uint(active.TargetId))
		ref := zoneRefFromKey(uint(zone_key))
		order.TargetZone = &ref
	case defs.OrderType_UnitAttackSpace:
		x, y := util.InvertPair(uint(active.TargetId))
		order.TargetSpace = &ai.Coordinate{X: x, Y: y}
	case defs.OrderType_UnitAttackUnit:
		if target, ok := v.units[uint64(active.TargetId)]; ok {
			view := v.unitViewShallow(target)
			order.TargetUnit = &view
		}
	case defs.OrderType_UnitAttackBuilding, defs.OrderType_UnitGarrison, defs.OrderType_UnitRepair, defs.OrderType_UnitRenew:
		if target, ok := v.buildings[uint64(active.TargetId)]; ok {
			view := v.buildingView(target)
			order.TargetBuilding = &view
		}
	}
	return order
}

func (v *aiView) unitLocation(u *snapUnit) ai.ZoneRef {
	if u.GarrisonedIn != nil {
		if b, ok := v.buildings[*u.GarrisonedIn]; ok {
			return zoneRefOf(b.Space, b.Zone)
		}
	}
	return zoneRefOf(u.Space, u.Zone)
}

// Doesn't populate CurrentOrder, so two units targeting each other can't recurse forever
func (v *aiView) unitViewShallow(u *snapUnit) ai.UnitView {
	kind := ai.UnitMilitary
	if u.UnitType == defs.UnitType_ECONOMIC {
		kind = ai.UnitEconomic
	}
	return ai.UnitView{
		InternalID:       u.InternalId,
		UnitID:           u.UnitId,
		Type:             ai.UnitType(u.UnitType),
		Kind:             kind,
		Location:         v.unitLocation(u),
		CurrentStamina:   u.CurrentStamina,
		Builds:           unitBuilds(u.UnitId),
		GarrisonedIn:     u.GarrisonedIn,
		Stance:           ai.UnitStance(u.Stance),
		InterruptCurrent: u.InterruptCurrent,
		AttackBack:       u.AttackBack,
		TargetPriority:   toAiTargetCategories(u.TargetPriority),
	}
}

func (v *aiView) unitView(u *snapUnit) ai.UnitView {
	view := v.unitViewShallow(u)
	view.CurrentOrder = v.currentOrder(u.ActiveOrders)
	if override, touched := v.assigned_units[u.InternalId]; touched {
		view.CurrentOrder = override
	}
	for _, o := range u.ActiveOrders {
		if kind, ok := toOrderKind(o.OrderType); ok && v.order_ids[o.InternalId] {
			view.ActiveOrders = append(view.ActiveOrders, ai.ActiveUnitOrder{ID: o.InternalId, Kind: kind})
		}
	}
	return view
}

func (v *aiView) producibles(b *snapBuilding) []ai.Producible {
	producibles := make([]ai.Producible, 0)
	for _, p := range defs.BuildingConfigs[b.BuildingId].Produces {
		switch p.Kind {
		case defs.ProducibleKind_UNIT:
			cost, _ := defs.UnitProductionCost(p.Id)
			producibles = append(producibles, ai.Producible{Kind: ai.ProducibleUnit, ID: p.Id, Cost: toCost(cost)})
		case defs.ProducibleKind_TECH:
			if _, researching_or_done := v.me.ResearchedTechs[p.Id]; researching_or_done {
				continue
			}
			producibles = append(producibles, ai.Producible{Kind: ai.ProducibleTech, ID: p.Id, Cost: toCost(defs.TechConfigs[p.Id].Cost)})
		}
	}
	return producibles
}

func (v *aiView) buildingView(b *snapBuilding) ai.BuildingView {
	config := defs.BuildingConfigs[b.BuildingId]
	return ai.BuildingView{
		InternalID:        b.InternalId,
		BuildingID:        b.BuildingId,
		Location:          zoneRefOf(b.Space, b.Zone),
		UnderConstruction: b.UnderConstruction,
		Idle:              !b.UnderConstruction && len(b.ActiveOrders) == 0 && len(config.Produces) > 0,
		Producibles:       v.producibles(b),
		GarrisonCount:     len(b.GarrisonedUnits),
		GarrisonCapacity:  b.GarrisonCapacity,
		Gatherable:        config.IsGatherable(),
		ResourcesLeft:     b.ResourcesLeft,
		Renewing:          b.Renewing,
		RenewCost:         toCost(config.Gather.Renew_cost),
		Health:            b.CombatStats.Health,
		MaxHealth:         float64(b.CombatStats.MaxHealth),
		CanAttack:         config.Attack_type != defs.AttackType_NONE,
	}
}

func (v *aiView) ownBuildingView(b *snapBuilding) ai.BuildingView {
	view := v.buildingView(b)
	view.AutoAttack = b.AutoAttack
	view.InterruptCurrent = b.InterruptCurrent
	view.TargetPriority = toAiTargetCategories(b.TargetPriority)
	for _, o := range b.ActiveOrders {
		if kind, ok := toBuildingOrderKind(o.OrderType); ok && v.order_ids[o.InternalId] {
			view.ActiveOrders = append(view.ActiveOrders, ai.ActiveBuildingOrder{ID: o.InternalId, Kind: kind, ItemID: uint32(o.TargetId)})
		}
	}
	return view
}

func (v *aiView) Units() []ai.UnitView {
	units := make([]ai.UnitView, 0, len(v.me.Units))
	for i := range v.me.Units {
		units = append(units, v.unitView(&v.me.Units[i]))
	}
	return sortUnitViews(units)
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
		foundations[location] = &ai.FoundationView{BuildingID: building_id, Location: location, Planned: true}
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

func (v *aiView) OwnedSpaces() int {
	owned := 0
	for _, space := range v.spaces {
		if owner, known := spaceOwner(space); known && owner == v.playerId() {
			owned++
		}
	}
	return owned
}

func (v *aiView) AllSpaces() []ai.SpaceInfo {
	infos := make([]ai.SpaceInfo, 0, len(v.spaces))
	for _, row := range v.game.Spaces {
		for _, space := range row {
			if space != nil {
				owner, _ := spaceOwner(space)
				infos = append(infos, ai.SpaceInfo{Space: toCoordinate(space.Coordinate), Vision: space.Visibility, Owner: owner})
			}
		}
	}
	return infos
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

func (v *aiView) InAttackRange(b ai.BuildingView, target ai.ZoneRef) bool {
	building, ok := v.buildings[b.InternalID]
	if !ok {
		return false
	}
	radius, ranged := building.AttackRange.SpaceRadius()
	if !ranged {
		return b.Location == target
	}
	return axialDistance(b.Location.Space, target.Space) <= int(radius)
}

func (v *aiView) VisibleEnemyUnits() []ai.UnitView {
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

// Returns the node's view, per-gatherer turn rate, and gatherer capacity
func (v *aiView) gatherableAt(entry snapZoneEntry, category ai.ResourceCategory) (ai.ResourceView, int, int, bool) {
	if r := entry.zone.Resource; r != nil {
		view := ai.ResourceView{InternalID: r.InternalId, Category: toAiCategory(defs.ResourceConfigs[r.ResourceId].Category), Location: entry.ref, AmountLeft: r.ResourcesLeft}
		return view, r.BaseGatherSpeed, r.GatherCapacity, r.ResourcesLeft > 0 && view.Category == category
	}
	b := entry.zone.Building
	if b == nil || b.PlayerId != v.playerId() || b.UnderConstruction || b.ResourcesLeft <= 0 {
		return ai.ResourceView{}, 0, 0, false
	}
	config := defs.BuildingConfigs[b.BuildingId]
	if !config.IsGatherable() || toAiCategory(config.Gather.Resource_category) != category {
		return ai.ResourceView{}, 0, 0, false
	}
	view := ai.ResourceView{InternalID: b.InternalId, Category: category, Location: entry.ref, AmountLeft: b.ResourcesLeft}
	return view, config.Gather.Base_gather_speed, config.Gather.Gather_capacity, true
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
	order, assigned := v.assigned_units[unit_id]
	if u, ok := v.units[unit_id]; ok && !assigned {
		order = v.currentOrder(u.ActiveOrders)
	}
	if order == nil || order.TargetResource == nil {
		return ai.ZoneRef{}, false
	}
	return order.TargetResource.Location, true
}

// Prefers the nearest node that its current gatherers plus this one won't empty within a turn.
func (v *aiView) NearestResource(from ai.ZoneRef, category ai.ResourceCategory, gatherer_id uint64) (ai.ResourceView, bool) {
	own_target, has_own_target := v.gatherTarget(gatherer_id)
	var best ai.ResourceView
	found, best_distance, best_saturated := false, -1, false
	for _, entry := range v.zones {
		view, speed, capacity, ok := v.gatherableAt(entry, category)
		if !ok {
			continue
		}
		count := v.gathererCount(entry.ref)
		if has_own_target && own_target == entry.ref {
			count--
		}
		if count >= capacity {
			continue
		}
		saturated := view.AmountLeft < float64((count+1)*speed)
		d := zoneDistance(from, entry.ref)
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
		if r := entry.zone.Resource; r != nil && r.ResourcesLeft > 0 && toAiCategory(defs.ResourceConfigs[r.ResourceId].Category) == category {
			resources = append(resources, ai.ResourceView{InternalID: r.InternalId, Category: category, Location: entry.ref, AmountLeft: r.ResourcesLeft})
		}
	}
	return resources
}

func (v *aiView) BuildCost(building_id uint32) ai.Cost {
	cost, _ := defs.BuildingProductionCost(building_id)
	return toCost(cost)
}

func (v *aiView) UnitCost(unit_id uint32) ai.Cost {
	cost, _ := defs.UnitProductionCost(unit_id)
	return toCost(cost)
}

func (v *aiView) TechCost(tech_id uint32) ai.Cost {
	return toCost(defs.TechConfigs[tech_id].Cost)
}

func (v *aiView) NearestUnexplored(from ai.ZoneRef) ([]ai.ZoneRef, bool) {
	var tied []ai.ZoneRef
	best_distance := -1
	for _, row := range v.game.Spaces {
		for _, space := range row {
			if space == nil || space.Visibility != defs.VisibilityUnexplored {
				continue
			}
			center := ai.ZoneRef{Space: toCoordinate(space.Coordinate)}
			d := axialDistance(from.Space, center.Space)
			if best_distance == -1 || d < best_distance {
				tied, best_distance = []ai.ZoneRef{center}, d
			} else if d == best_distance {
				tied = append(tied, center)
			}
		}
	}
	return tied, len(tied) > 0
}

// Mirrors the build rule against snapshot ownership: a space I own, or an unowned one bordering one I own
func (v *aiView) buildable(space *snapSpace) bool {
	owner, known := spaceOwner(space)
	if !known || owner != -1 {
		return known && owner == v.playerId()
	}
	for _, direction := range game_utils.AxialDirectionVectors() {
		adjacent := v.spaces[ai.Coordinate{X: space.Coordinate.X + direction.X, Y: space.Coordinate.Y + direction.Y}]
		if adj_owner, adj_known := spaceOwner(adjacent); adj_known && adj_owner == v.playerId() {
			return true
		}
	}
	return false
}

func (v *aiView) NearestBuildSite(from ai.ZoneRef, _ uint32) (ai.ZoneRef, bool) {
	foundations := v.foundationsByLocation()
	var best ai.ZoneRef
	found, best_distance := false, -1
	for _, entry := range v.zones {
		if entry.zone.Resource != nil || entry.zone.Building != nil || foundations[entry.ref] != nil || !v.buildable(entry.space) {
			continue
		}
		d := zoneDistance(from, entry.ref)
		if !found || d < best_distance || (d == best_distance && zoneKey(entry.ref) < zoneKey(best)) {
			best, best_distance, found = entry.ref, d, true
		}
	}
	return best, found
}

func (v *aiView) assignUnit(id uint64, order *ai.CurrentOrder) {
	if v.gather_counts != nil {
		if previous, ok := v.gatherTarget(id); ok {
			v.gather_counts[previous]--
		}
		if order.TargetResource != nil {
			v.gather_counts[order.TargetResource.Location]++
		}
	}
	v.assigned_units[id] = order
}

func (v *aiView) claimBuilding(id uint64) {
	v.claimed_buildings[id] = true
}
