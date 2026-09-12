package ai

// Axial hex coordinate; space- or zone-level depending on context.
type Coordinate struct {
	X, Y int
}

type ZoneRef struct {
	Space Coordinate
	Zone  Coordinate
}

type ResourceCategory uint8

const (
	ResourceFood ResourceCategory = iota
	ResourceWood
	ResourceStone
	ResourceGold
)

type UnitKind uint8

const (
	UnitEconomic UnitKind = iota
	UnitMilitary
)

type UnitView struct {
	InternalID     uint64
	UnitID         uint32
	Kind           UnitKind
	Location       ZoneRef
	CurrentStamina int
	CurrentOrder   *CurrentOrder
	Builds         []Producible
	GarrisonedIn   *uint64
}

// Mirrors the unit-applicable subset of risq.OrderType, so config can filter eligible units by task
type OrderKind uint8

const (
	OrderKindMove OrderKind = iota
	OrderKindGather
	OrderKindBuild
	OrderKindRepair
	OrderKindAttackSpace
	OrderKindAttackZone
	OrderKindAttackUnit
	OrderKindAttackBuilding
	OrderKindDefend
	OrderKindGarrison
	OrderKindUngarrison
	OrderKindDelete
)

var orderKindNames = map[string]OrderKind{
	"move":            OrderKindMove,
	"gather":          OrderKindGather,
	"build":           OrderKindBuild,
	"repair":          OrderKindRepair,
	"attack_space":    OrderKindAttackSpace,
	"attack_zone":     OrderKindAttackZone,
	"attack_unit":     OrderKindAttackUnit,
	"attack_building": OrderKindAttackBuilding,
	"defend":          OrderKindDefend,
	"garrison":        OrderKindGarrison,
	"ungarrison":      OrderKindUngarrison,
	"delete":          OrderKindDelete,
}

type CurrentOrder struct {
	Kind           OrderKind
	TargetZone     *ZoneRef
	TargetSpace    *Coordinate
	TargetResource *ResourceView
	TargetUnit     *UnitView
	TargetBuilding *BuildingView
}

type ResourceView struct {
	InternalID uint64
	Category   ResourceCategory
	Location   ZoneRef
	AmountLeft float64
}

type ProducibleKind uint8

const (
	ProducibleUnit ProducibleKind = iota
	ProducibleTech
	ProducibleBuilding
)

type Cost struct {
	Food, Wood, Stone, Gold float64
}

// Something a building can currently be ordered to make (already-researched techs excluded).
type Producible struct {
	Kind ProducibleKind
	ID   uint32
	Cost Cost
}

type BuildingView struct {
	InternalID       uint64
	BuildingID       uint32
	Location         ZoneRef
	UnderConstruction bool
	Idle             bool
	Producibles      []Producible
	GarrisonCount    int
	GarrisonCapacity int
}

// Mirrors risq.OrderFromFrontend; Player_id is stamped on by risq.
type Order struct {
	Subjects            []uint64
	OrderType           uint8
	TargetID            int64
	ClearPreviousOrders bool
}

// Read-only per-player snapshot, gated by the same fog-of-war the frontend shows a human.
type View interface {
	Nickname() string
	Units() []UnitView
	IdleUnits() []UnitView
	EligibleUnits(kinds ...OrderKind) []UnitView
	Buildings() []BuildingView
	IdleBuildings() []BuildingView
	Resource(category ResourceCategory) float64
	Population() (current, limit int)

	VisibleEnemyUnits() []UnitView
	VisibleEnemyBuildings() []BuildingView

	NearestResource(from ZoneRef, category ResourceCategory) (ResourceView, bool)
	NearestBuildSite(from ZoneRef, building_id uint32) (ZoneRef, bool)
	NearestUnexplored(from ZoneRef) ([]ZoneRef, bool)
	TurnNumber() int
	AllEnemiesFound() bool
	BuildCost(building_id uint32) Cost
	UnitCost(unit_id uint32) Cost
	TechCost(tech_id uint32) Cost
	RandomIntn(n int) int

	MoveOrder(u UnitView, target ZoneRef, clear_previous bool) Order
	GatherOrder(u UnitView, target ResourceView, clear_previous bool) Order
	BuildOrder(u UnitView, building_id uint32, target ZoneRef, clear_previous bool) Order
	RepairOrder(u UnitView, target BuildingView, clear_previous bool) Order
	AttackUnitOrder(u UnitView, target UnitView, clear_previous bool) Order
	AttackBuildingOrder(u UnitView, target BuildingView, clear_previous bool) Order
	AttackSpaceOrder(u UnitView, target Coordinate, clear_previous bool) Order
	AttackZoneOrder(u UnitView, target ZoneRef, clear_previous bool) Order
	GarrisonOrder(u UnitView, target BuildingView, clear_previous bool) Order
	UngarrisonOrder(u UnitView, clear_previous bool) Order
	DeleteUnitOrder(u UnitView) Order
	CreateUnitOrder(b BuildingView, unit_id uint32) Order
	ResearchOrder(b BuildingView, tech_id uint32) Order
}
