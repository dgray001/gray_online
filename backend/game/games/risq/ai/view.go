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

type UnitType uint8

const (
	UnitTypeNone UnitType = iota
	UnitTypeEconomic
	UnitTypeInfantry
	UnitTypeArcher
	UnitTypeCavalry
)

var unitTypeNames = map[string]UnitType{
	"economic": UnitTypeEconomic,
	"infantry": UnitTypeInfantry,
	"archer":   UnitTypeArcher,
	"cavalry":  UnitTypeCavalry,
}

type UnitStance uint8

const (
	UnitStanceNone UnitStance = iota
	UnitStancePassive
	UnitStanceAggressive
	UnitStanceDefensive
	UnitStanceStandGround
)

var unitStanceNames = map[string]UnitStance{
	"passive":      UnitStancePassive,
	"aggressive":   UnitStanceAggressive,
	"defensive":    UnitStanceDefensive,
	"stand_ground": UnitStanceStandGround,
}

type TargetCategory uint8

const (
	TargetCategoryNone TargetCategory = iota
	TargetCategoryEconomic
	TargetCategoryMilitary
	TargetCategoryBuilding
)

var targetCategoryNames = map[string]TargetCategory{
	"economic": TargetCategoryEconomic,
	"military": TargetCategoryMilitary,
	"building": TargetCategoryBuilding,
}

type UnitView struct {
	InternalID       uint64
	UnitID           uint32
	Type             UnitType
	Kind             UnitKind
	Location         ZoneRef
	CurrentStamina   int
	CurrentOrder     *CurrentOrder
	Builds           []Producible
	GarrisonedIn     *uint64
	Stance           UnitStance
	InterruptCurrent bool
	AttackBack       bool
	TargetPriority   []TargetCategory
	ActiveOrders     []ActiveUnitOrder
}

type UnitBehavior struct {
	Subjects         []uint64
	Stance           *UnitStance
	InterruptCurrent *bool
	AttackBack       *bool
	TargetPriority   []TargetCategory
}

type Decision struct {
	Orders            []Order
	Behaviors         []UnitBehavior
	BuildingBehaviors []BuildingBehavior
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
	OrderKindGarrison
	OrderKindUngarrison
	OrderKindDelete
	OrderKindRenew
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
	"garrison":        OrderKindGarrison,
	"ungarrison":      OrderKindUngarrison,
	"delete":          OrderKindDelete,
	"renew":           OrderKindRenew,
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

func (c Cost) of(category ResourceCategory) float64 {
	switch category {
	case ResourceFood:
		return c.Food
	case ResourceWood:
		return c.Wood
	case ResourceStone:
		return c.Stone
	default:
		return c.Gold
	}
}

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
	Idle              bool
	Producibles       []Producible
	GarrisonCount     int
	GarrisonCapacity  int
	Gatherable        bool
	ResourcesLeft     float64
	Renewing          bool
	RenewCost         Cost
	Health            float64
	MaxHealth         float64
	CanAttack         bool
	ActiveOrders      []ActiveBuildingOrder
	AutoAttack        bool
	InterruptCurrent  bool
	TargetPriority    []TargetCategory
}

type BuildingBehavior struct {
	Subjects         []uint64
	AutoAttack       *bool
	InterruptCurrent *bool
	TargetPriority   []TargetCategory
}

type BuildingOrderKind uint8

const (
	BuildingOrderCreate BuildingOrderKind = iota
	BuildingOrderResearch
	BuildingOrderDelete
	BuildingOrderAttackUnit
	BuildingOrderAttackBuilding
)

var buildingOrderKindNames = map[string]BuildingOrderKind{
	"create":          BuildingOrderCreate,
	"research":        BuildingOrderResearch,
	"delete":          BuildingOrderDelete,
	"attack_unit":     BuildingOrderAttackUnit,
	"attack_building": BuildingOrderAttackBuilding,
}

type ActiveBuildingOrder struct {
	ID     uint64
	Kind   BuildingOrderKind
	ItemID uint32
}

type ActiveUnitOrder struct {
	ID   uint64
	Kind OrderKind
}

type FoundationView struct {
	BuildingID uint32
	Location   ZoneRef
	Planned    bool
	Builders   int
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
	Foundations() []FoundationView

	VisibleEnemyUnits() []UnitView
	VisibleEnemyBuildings() []BuildingView

	NearestResource(from ZoneRef, category ResourceCategory) (ResourceView, bool)
	NearestBuildSite(from ZoneRef, building_id uint32) (ZoneRef, bool)
	NearestUnexplored(from ZoneRef) ([]ZoneRef, bool)
	TurnNumber() int
	NumPlayers() int
	EnemiesFound() int
	OwnedSpaces() int
	Score() int
	BestEnemyScore() int
	TechResearched(tech_id uint32) bool
	InAttackRange(b BuildingView, target ZoneRef) bool
	BuildCost(building_id uint32) Cost
	UnitCost(unit_id uint32) Cost
	TechCost(tech_id uint32) Cost
	RandomIntn(n int) int

	MoveOrder(u UnitView, target ZoneRef, clear_previous bool) Order
	GatherOrder(u UnitView, target ResourceView, clear_previous bool) Order
	BuildOrder(u UnitView, building_id uint32, target ZoneRef, clear_previous bool) Order
	RepairOrder(u UnitView, target BuildingView, clear_previous bool) Order
	RenewOrder(u UnitView, target BuildingView, clear_previous bool) Order
	AttackUnitOrder(u UnitView, target UnitView, clear_previous bool) Order
	AttackBuildingOrder(u UnitView, target BuildingView, clear_previous bool) Order
	AttackSpaceOrder(u UnitView, target Coordinate, clear_previous bool) Order
	AttackZoneOrder(u UnitView, target ZoneRef, clear_previous bool) Order
	GarrisonOrder(u UnitView, target BuildingView, clear_previous bool) Order
	UngarrisonOrder(u UnitView, clear_previous bool) Order
	DeleteUnitOrder(u UnitView) Order
	CreateUnitOrder(b BuildingView, unit_id uint32) Order
	ResearchOrder(b BuildingView, tech_id uint32) Order
	DeleteBuildingOrder(b BuildingView) Order
	BuildingAttackUnitOrder(b BuildingView, target UnitView) Order
	BuildingAttackBuildingOrder(b BuildingView, target BuildingView) Order
	CancelFoundationOrder(f FoundationView) Order
	CancelOrder(order_id uint64) Order
}
