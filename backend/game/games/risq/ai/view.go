package ai

// Axial hex coordinate; space- or zone-level depending on context.
type Coordinate struct{ X, Y int }

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
	Idle           bool // no active orders
}

type ProducibleKind uint8

const (
	ProducibleUnit ProducibleKind = iota
	ProducibleTech
)

type Cost struct{ Food, Wood, Stone, Gold float64 }

// Something a building can currently be ordered to make (already-researched techs excluded).
type Producible struct {
	Kind ProducibleKind
	ID   uint32
	Cost Cost
}

type BuildingView struct {
	InternalID        uint64
	BuildingID        uint32
	Location          ZoneRef
	UnderConstruction bool
	Idle              bool // can produce something and isn't
	Producibles       []Producible
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
	Buildings() []BuildingView
	IdleBuildings() []BuildingView
	Resource(category ResourceCategory) float64
	Population() (current, limit int)

	VisibleEnemyUnits() []UnitView
	VisibleEnemyBuildings() []BuildingView

	NearestResource(from ZoneRef, category ResourceCategory) (ZoneRef, bool)
	NearestBuildSite(from ZoneRef, building_id uint32) (ZoneRef, bool)
	NearestUnexplored(from ZoneRef) (ZoneRef, bool)
	BuildCost(building_id uint32) Cost

	MoveOrder(u UnitView, target ZoneRef, clear_previous bool) Order
	GatherOrder(u UnitView, target ZoneRef, clear_previous bool) Order
	BuildOrder(u UnitView, building_id uint32, target ZoneRef, clear_previous bool) Order
	RepairOrder(u UnitView, target_building_id uint64, clear_previous bool) Order
	AttackUnitOrder(u UnitView, target_unit_id uint64, clear_previous bool) Order
	AttackBuildingOrder(u UnitView, target_building_id uint64, clear_previous bool) Order
	DeleteUnitOrder(u UnitView) Order
	CreateUnitOrder(b BuildingView, unit_id uint32) Order
	ResearchOrder(b BuildingView, tech_id uint32) Order
}
