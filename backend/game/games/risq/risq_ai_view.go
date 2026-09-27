package risq

import (
	"sort"

	"github.com/dgray001/gray_online/game/game_utils"
	"github.com/dgray001/gray_online/game/games/risq/ai"
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

func toCoordinate(c snapCoordinate) ai.Coordinate {
	return ai.Coordinate{X: c.X, Y: c.Y}
}

func zoneRefOf(space snapCoordinate, zone snapCoordinate) ai.ZoneRef {
	return ai.ZoneRef{Space: toCoordinate(space), Zone: toCoordinate(zone)}
}

func spaceKey(c ai.Coordinate) uint {
	return util.Pair(c.X, c.Y)
}

func zoneKey(ref ai.ZoneRef) uint {
	return util.Pair(int(spaceKey(ref.Space)), int(util.Pair(ref.Zone.X, ref.Zone.Y)))
}

func zoneRefFromKey(k uint) ai.ZoneRef {
	space_key, local_key := util.InvertPair(k)
	x, y := util.InvertPair(uint(space_key))
	i, j := util.InvertPair(uint(local_key))
	return ai.ZoneRef{Space: ai.Coordinate{X: x, Y: y}, Zone: ai.Coordinate{X: i, Y: j}}
}

func axialDistance(a ai.Coordinate, b ai.Coordinate) int {
	return int(game_utils.AxialDistance(game_utils.Coordinate2D{X: a.X, Y: a.Y}, game_utils.Coordinate2D{X: b.X, Y: b.Y}))
}

// Zones within a space are one step apart unless opposite; crossing into another space costs more
func zoneDistance(a ai.ZoneRef, b ai.ZoneRef) int {
	if a.Space == b.Space {
		return axialDistance(a.Zone, b.Zone)
	}
	return axialDistance(a.Space, b.Space) * 6
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

func spaceOwner(space *snapSpace) (int, bool) {
	if space == nil || space.Ownership == nil {
		return -1, false
	}
	return *space.Ownership, true
}

func toCost(c RisqResourceCost) ai.Cost {
	return ai.Cost{Food: c.food, Wood: c.wood, Stone: c.stone, Gold: c.gold}
}

func toAiCategory(c RisqResourceCategory) ai.ResourceCategory {
	switch c {
	case RisqResourceCategory_FOOD:
		return ai.ResourceFood
	case RisqResourceCategory_WOOD:
		return ai.ResourceWood
	case RisqResourceCategory_STONE:
		return ai.ResourceStone
	default:
		return ai.ResourceGold
	}
}

func toOrderKind(order_type OrderType) (ai.OrderKind, bool) {
	switch order_type {
	case OrderType_UnitMoveSpace, OrderType_UnitMoveZone:
		return ai.OrderKindMove, true
	case OrderType_UnitGather:
		return ai.OrderKindGather, true
	case OrderType_UnitBuild:
		return ai.OrderKindBuild, true
	case OrderType_UnitRepair:
		return ai.OrderKindRepair, true
	case OrderType_UnitAttackSpace:
		return ai.OrderKindAttackSpace, true
	case OrderType_UnitAttackZone:
		return ai.OrderKindAttackZone, true
	case OrderType_UnitAttackUnit, OrderType_UnitAutoAttackUnit:
		return ai.OrderKindAttackUnit, true
	case OrderType_UnitAttackBuilding:
		return ai.OrderKindAttackBuilding, true
	case OrderType_UnitGarrison:
		return ai.OrderKindGarrison, true
	case OrderType_UnitUngarrison:
		return ai.OrderKindUngarrison, true
	case OrderType_UnitDelete:
		return ai.OrderKindDelete, true
	case OrderType_UnitRenew:
		return ai.OrderKindRenew, true
	default:
		return 0, false
	}
}

func toBuildingOrderKind(order_type OrderType) (ai.BuildingOrderKind, bool) {
	switch order_type {
	case OrderType_BuildingCreate:
		return ai.BuildingOrderCreate, true
	case OrderType_BuildingResearch:
		return ai.BuildingOrderResearch, true
	case OrderType_BuildingDelete:
		return ai.BuildingOrderDelete, true
	case OrderType_BuildingAttackUnit:
		return ai.BuildingOrderAttackUnit, true
	case OrderType_BuildingAttackBuilding:
		return ai.BuildingOrderAttackBuilding, true
	}
	return 0, false
}

func toAiTargetCategories(priority []int) []ai.TargetCategory {
	categories := make([]ai.TargetCategory, len(priority))
	for i, c := range priority {
		categories[i] = ai.TargetCategory(c)
	}
	return categories
}

func unitBuilds(unit_id uint32) []ai.Producible {
	builds := make([]ai.Producible, 0)
	for _, p := range unitConfigs[unit_id].builds {
		if p.kind != ProducibleKind_BUILDING {
			continue
		}
		cost, _ := buildingProductionCost(p.id)
		builds = append(builds, ai.Producible{Kind: ai.ProducibleBuilding, ID: p.id, Cost: toCost(cost)})
	}
	return builds
}

func sortUnitViews(units []ai.UnitView) []ai.UnitView {
	sort.Slice(units, func(i, j int) bool { return units[i].InternalID < units[j].InternalID })
	return units
}

func sortBuildingViews(buildings []ai.BuildingView) []ai.BuildingView {
	sort.Slice(buildings, func(i, j int) bool { return buildings[i].InternalID < buildings[j].InternalID })
	return buildings
}

func zoneRefLess(a ai.ZoneRef, b ai.ZoneRef) bool {
	ka := [4]int{a.Space.X, a.Space.Y, a.Zone.X, a.Zone.Y}
	kb := [4]int{b.Space.X, b.Space.Y, b.Zone.X, b.Zone.Y}
	for i := range ka {
		if ka[i] != kb[i] {
			return ka[i] < kb[i]
		}
	}
	return false
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
		return ai.ResourceView{InternalID: r.InternalId, Category: toAiCategory(resourceConfigs[r.ResourceId].category), Location: ref, AmountLeft: r.ResourcesLeft}, true
	}
	if b := zone.Building; b != nil && buildingConfigs[b.BuildingId].isGatherable() {
		return ai.ResourceView{InternalID: b.InternalId, Category: toAiCategory(buildingConfigs[b.BuildingId].gather.resource_category), Location: ref, AmountLeft: b.ResourcesLeft}, true
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
	case OrderType_UnitMoveZone, OrderType_UnitAttackZone:
		ref := zoneRefFromKey(uint(active.TargetId))
		order.TargetZone = &ref
	case OrderType_UnitGather:
		if resource, ok := v.gatherTargetAt(zoneRefFromKey(uint(active.TargetId))); ok {
			order.TargetResource = &resource
		}
	case OrderType_UnitBuild:
		_, zone_key := util.InvertPair(uint(active.TargetId))
		ref := zoneRefFromKey(uint(zone_key))
		order.TargetZone = &ref
	case OrderType_UnitAttackSpace:
		x, y := util.InvertPair(uint(active.TargetId))
		order.TargetSpace = &ai.Coordinate{X: x, Y: y}
	case OrderType_UnitAttackUnit:
		if target, ok := v.units[uint64(active.TargetId)]; ok {
			view := v.unitViewShallow(target)
			order.TargetUnit = &view
		}
	case OrderType_UnitAttackBuilding, OrderType_UnitGarrison, OrderType_UnitRepair, OrderType_UnitRenew:
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
	if u.UnitType == UnitType_ECONOMIC {
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
	for _, p := range buildingConfigs[b.BuildingId].produces {
		switch p.kind {
		case ProducibleKind_UNIT:
			cost, _ := unitProductionCost(p.id)
			producibles = append(producibles, ai.Producible{Kind: ai.ProducibleUnit, ID: p.id, Cost: toCost(cost)})
		case ProducibleKind_TECH:
			if _, researching_or_done := v.me.ResearchedTechs[p.id]; researching_or_done {
				continue
			}
			producibles = append(producibles, ai.Producible{Kind: ai.ProducibleTech, ID: p.id, Cost: toCost(techConfigs[p.id].cost)})
		}
	}
	return producibles
}

func (v *aiView) buildingView(b *snapBuilding) ai.BuildingView {
	config := buildingConfigs[b.BuildingId]
	return ai.BuildingView{
		InternalID:        b.InternalId,
		BuildingID:        b.BuildingId,
		Location:          zoneRefOf(b.Space, b.Zone),
		UnderConstruction: b.UnderConstruction,
		Idle:              !b.UnderConstruction && len(b.ActiveOrders) == 0 && len(config.produces) > 0,
		Producibles:       v.producibles(b),
		GarrisonCount:     len(b.GarrisonedUnits),
		GarrisonCapacity:  b.GarrisonCapacity,
		Gatherable:        config.isGatherable(),
		ResourcesLeft:     b.ResourcesLeft,
		Renewing:          b.Renewing,
		RenewCost:         toCost(config.gather.renew_cost),
		Health:            b.CombatStats.Health,
		MaxHealth:         float64(b.CombatStats.MaxHealth),
		CanAttack:         config.attack_type != AttackType_NONE,
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
	radius, ranged := building.AttackRange.spaceRadius()
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
		view := ai.ResourceView{InternalID: r.InternalId, Category: toAiCategory(resourceConfigs[r.ResourceId].category), Location: entry.ref, AmountLeft: r.ResourcesLeft}
		return view, r.BaseGatherSpeed, r.GatherCapacity, r.ResourcesLeft > 0 && view.Category == category
	}
	b := entry.zone.Building
	if b == nil || b.PlayerId != v.playerId() || b.UnderConstruction || b.ResourcesLeft <= 0 {
		return ai.ResourceView{}, 0, 0, false
	}
	config := buildingConfigs[b.BuildingId]
	if !config.isGatherable() || toAiCategory(config.gather.resource_category) != category {
		return ai.ResourceView{}, 0, 0, false
	}
	view := ai.ResourceView{InternalID: b.InternalId, Category: category, Location: entry.ref, AmountLeft: b.ResourcesLeft}
	return view, config.gather.base_gather_speed, config.gather.gather_capacity, true
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
		if r := entry.zone.Resource; r != nil && r.ResourcesLeft > 0 && toAiCategory(resourceConfigs[r.ResourceId].category) == category {
			resources = append(resources, ai.ResourceView{InternalID: r.InternalId, Category: category, Location: entry.ref, AmountLeft: r.ResourcesLeft})
		}
	}
	return resources
}

func (v *aiView) BuildCost(building_id uint32) ai.Cost {
	cost, _ := buildingProductionCost(building_id)
	return toCost(cost)
}

func (v *aiView) UnitCost(unit_id uint32) ai.Cost {
	cost, _ := unitProductionCost(unit_id)
	return toCost(cost)
}

func (v *aiView) TechCost(tech_id uint32) ai.Cost {
	return toCost(techConfigs[tech_id].cost)
}

func (v *aiView) NearestUnexplored(from ai.ZoneRef) ([]ai.ZoneRef, bool) {
	var tied []ai.ZoneRef
	best_distance := -1
	for _, row := range v.game.Spaces {
		for _, space := range row {
			if space == nil || space.Visibility != VisibilityUnexplored {
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

func unitOrder(u ai.UnitView, order_type OrderType, target_id int64, clear_previous bool) ai.Order {
	return ai.Order{Subjects: []uint64{u.InternalID}, OrderType: uint8(order_type), TargetID: target_id, ClearPreviousOrders: clear_previous}
}

func buildingOrder(b ai.BuildingView, order_type OrderType, target_id int64) ai.Order {
	return ai.Order{Subjects: []uint64{b.InternalID}, OrderType: uint8(order_type), TargetID: target_id}
}

func (v *aiView) MoveOrder(u ai.UnitView, target ai.ZoneRef, clear_previous bool) ai.Order {
	v.assignUnit(u.InternalID, &ai.CurrentOrder{Kind: ai.OrderKindMove, TargetZone: &target})
	return unitOrder(u, OrderType_UnitMoveZone, int64(zoneKey(target)), clear_previous)
}

func (v *aiView) GatherOrder(u ai.UnitView, target ai.ResourceView, clear_previous bool) ai.Order {
	v.assignUnit(u.InternalID, &ai.CurrentOrder{Kind: ai.OrderKindGather, TargetResource: &target})
	return unitOrder(u, OrderType_UnitGather, int64(zoneKey(target.Location)), clear_previous)
}

func (v *aiView) BuildOrder(u ai.UnitView, building_id uint32, target ai.ZoneRef, clear_previous bool) ai.Order {
	v.assignUnit(u.InternalID, &ai.CurrentOrder{Kind: ai.OrderKindBuild, TargetZone: &target})
	if !v.occupied(target) && !v.planned[target] {
		v.ordered_foundations[target] = building_id
	}
	return unitOrder(u, OrderType_UnitBuild, int64(util.Pair(int(building_id), int(zoneKey(target)))), clear_previous)
}

func (v *aiView) RepairOrder(u ai.UnitView, target ai.BuildingView, clear_previous bool) ai.Order {
	v.assignUnit(u.InternalID, &ai.CurrentOrder{Kind: ai.OrderKindRepair, TargetBuilding: &target})
	return unitOrder(u, OrderType_UnitRepair, int64(target.InternalID), clear_previous)
}

func (v *aiView) RenewOrder(u ai.UnitView, target ai.BuildingView, clear_previous bool) ai.Order {
	v.assignUnit(u.InternalID, &ai.CurrentOrder{Kind: ai.OrderKindRenew, TargetBuilding: &target})
	return unitOrder(u, OrderType_UnitRenew, int64(target.InternalID), clear_previous)
}

func (v *aiView) AttackUnitOrder(u ai.UnitView, target ai.UnitView, clear_previous bool) ai.Order {
	v.assignUnit(u.InternalID, &ai.CurrentOrder{Kind: ai.OrderKindAttackUnit, TargetUnit: &target})
	return unitOrder(u, OrderType_UnitAttackUnit, int64(target.InternalID), clear_previous)
}

func (v *aiView) AttackBuildingOrder(u ai.UnitView, target ai.BuildingView, clear_previous bool) ai.Order {
	v.assignUnit(u.InternalID, &ai.CurrentOrder{Kind: ai.OrderKindAttackBuilding, TargetBuilding: &target})
	return unitOrder(u, OrderType_UnitAttackBuilding, int64(target.InternalID), clear_previous)
}

func (v *aiView) AttackSpaceOrder(u ai.UnitView, target ai.Coordinate, clear_previous bool) ai.Order {
	v.assignUnit(u.InternalID, &ai.CurrentOrder{Kind: ai.OrderKindAttackSpace, TargetSpace: &target})
	return unitOrder(u, OrderType_UnitAttackSpace, int64(spaceKey(target)), clear_previous)
}

func (v *aiView) AttackZoneOrder(u ai.UnitView, target ai.ZoneRef, clear_previous bool) ai.Order {
	v.assignUnit(u.InternalID, &ai.CurrentOrder{Kind: ai.OrderKindAttackZone, TargetZone: &target})
	return unitOrder(u, OrderType_UnitAttackZone, int64(zoneKey(target)), clear_previous)
}

func (v *aiView) GarrisonOrder(u ai.UnitView, target ai.BuildingView, clear_previous bool) ai.Order {
	v.assignUnit(u.InternalID, &ai.CurrentOrder{Kind: ai.OrderKindGarrison, TargetBuilding: &target})
	return unitOrder(u, OrderType_UnitGarrison, int64(target.InternalID), clear_previous)
}

func (v *aiView) UngarrisonOrder(u ai.UnitView, clear_previous bool) ai.Order {
	v.assignUnit(u.InternalID, &ai.CurrentOrder{Kind: ai.OrderKindUngarrison})
	return unitOrder(u, OrderType_UnitUngarrison, 0, clear_previous)
}

func (v *aiView) DeleteUnitOrder(u ai.UnitView) ai.Order {
	v.assignUnit(u.InternalID, &ai.CurrentOrder{Kind: ai.OrderKindDelete})
	return unitOrder(u, OrderType_UnitDelete, 0, true)
}

func (v *aiView) CreateUnitOrder(b ai.BuildingView, unit_id uint32) ai.Order {
	v.claimBuilding(b.InternalID)
	return buildingOrder(b, OrderType_BuildingCreate, int64(unit_id))
}

func (v *aiView) ResearchOrder(b ai.BuildingView, tech_id uint32) ai.Order {
	v.claimBuilding(b.InternalID)
	return buildingOrder(b, OrderType_BuildingResearch, int64(tech_id))
}

func (v *aiView) DeleteBuildingOrder(b ai.BuildingView) ai.Order {
	v.claimBuilding(b.InternalID)
	return buildingOrder(b, OrderType_BuildingDelete, 0)
}

func (v *aiView) BuildingAttackUnitOrder(b ai.BuildingView, target ai.UnitView) ai.Order {
	v.claimBuilding(b.InternalID)
	return buildingOrder(b, OrderType_BuildingAttackUnit, int64(target.InternalID))
}

func (v *aiView) BuildingAttackBuildingOrder(b ai.BuildingView, target ai.BuildingView) ai.Order {
	v.claimBuilding(b.InternalID)
	return buildingOrder(b, OrderType_BuildingAttackBuilding, int64(target.InternalID))
}

func (v *aiView) CancelOrder(order_id uint64) ai.Order {
	return ai.Order{OrderType: uint8(OrderType_CancelOrder), TargetID: int64(order_id)}
}

func (v *aiView) CancelFoundationOrder(f ai.FoundationView) ai.Order {
	v.cancelled_foundations[f.Location] = true
	return ai.Order{OrderType: uint8(OrderType_CancelFoundation), TargetID: int64(zoneKey(f.Location))}
}
