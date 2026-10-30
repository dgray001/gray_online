package ai

import "github.com/dgray001/gray_online/util"

// Axial hex coordinate; space- or zone-level depending on context.
type Coordinate struct {
	X, Y int
}

type SpaceInfo struct {
	TerrainType        uint8
	IntraCost          uint
	InterCost          uint
	BuildSpeedModifier float64
	Passable           bool

	UnitCount  *int
	Space      Coordinate
	Vision     uint8
	Owner      int
	GoldIncome float64
}

type RegionInfo struct {
	Name          string
	GoldBonus     float64
	Owner         int
	Spaces        []Coordinate
	HasUnexplored bool
}

type ZoneRef struct {
	Space Coordinate
	Zone  Coordinate
}

// What is known of one zone; BuildingPlayer is -1 with no building
type ZoneInfo struct {
	Location       ZoneRef
	HasResource    bool
	BuildingPlayer int
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
	UnitTypeSiege
)

var unitTypeNames = map[string]UnitType{
	"economic": UnitTypeEconomic,
	"infantry": UnitTypeInfantry,
	"archer":   UnitTypeArcher,
	"cavalry":  UnitTypeCavalry,
	"siege":    UnitTypeSiege,
}

// "military" stands for every non-economic unit type
func unitTypesNamed(name string) []UnitType {
	if name == "military" {
		return []UnitType{UnitTypeInfantry, UnitTypeArcher, UnitTypeCavalry, UnitTypeSiege}
	}
	if unit_type, ok := unitTypeNames[name]; ok {
		return []UnitType{unit_type}
	}
	return nil
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

type AttackRange uint8

const (
	AttackRangeNone AttackRange = iota
	AttackRangeZone
	AttackRangeWall
	AttackRangeSpace
	AttackRangeTwozone
	AttackRangeThreezone
	AttackRangeAdjacent
	AttackRangeFivezone
	AttackRangeSecondary
)

func (r AttackRange) SpaceReach() int {
	switch r {
	case AttackRangeWall, AttackRangeSpace, AttackRangeTwozone, AttackRangeThreezone:
		return 0
	case AttackRangeAdjacent, AttackRangeFivezone:
		return 1
	case AttackRangeSecondary:
		return 2
	}
	return -1
}

type UnitView struct {
	AttackRange      AttackRange
	CanAttack        bool
	MaxStamina       int
	InternalID       uint64
	PlayerID         int
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
	Health           float64
	MaxHealth        float64
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
	OrderKindAutoAttackUnit
	OrderKindAutoAttackBuilding
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

	"auto_attack_unit":     OrderKindAutoAttackUnit,
	"auto_attack_building": OrderKindAutoAttackBuilding,
}

type CurrentOrder struct {
	TargetUID      uint64
	Kind           OrderKind
	TargetZone     *ZoneRef
	TargetSpace    *Coordinate
	TargetResource *ResourceView
	TargetUnit     *UnitView
	TargetBuilding *BuildingView
}

type ResourceView struct {
	InternalID     uint64
	Category       ResourceCategory
	Location       ZoneRef
	AmountLeft     float64
	GatherSpeed    int
	GatherCapacity int
	IsBuilding     bool
}

type ProducibleKind uint8

const (
	ProducibleNone ProducibleKind = iota
	ProducibleUnit
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
	UnitType    UnitType
	Kind        ProducibleKind
	ID          uint32
	Cost        Cost
	StaminaCost int
}

type BuildingView struct {
	AttackRange       AttackRange
	MaxStamina        int
	Visible           bool
	InternalID        uint64
	PlayerID          int
	BuildingID        uint32
	Location          ZoneRef
	UnderConstruction bool
	Idle              bool
	PlannedProduction int // create and research orders already emitted this turn
	Producibles       []Producible
	GarrisonCount     int
	GarrisonCapacity  int
	Gatherable        bool
	ResourceCategory  ResourceCategory
	ResourcesLeft     float64
	GatherSpeed       int
	GatherCapacity    int
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
	BuildingID      uint32
	Location        ZoneRef
	Planned         bool
	Builders        int
	OrderedThisTurn bool
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
	OrderQueue() []OrderEntry
	Logger() *util.Logger
	// changes whenever an emitted order changes what the view reports
	Revision() uint64
	Nickname() string
	Units() []UnitView
	AllUnits() []UnitView
	AllBuildings() []BuildingView
	Issue(request IssueRequest) Order
	IdleUnits() []UnitView
	Buildings() []BuildingView
	IdleBuildings() []BuildingView
	Resource(category ResourceCategory) float64
	Population() (current, limit int)
	// the game's population ceiling, which no amount of housing can raise
	MaxPopulation() int
	Foundations() []FoundationView

	VisibleEnemyUnits() []UnitView
	VisibleEnemyBuildings() []BuildingView
	// visible enemy buildings plus the last-seen ones in fogged spaces
	KnownEnemyBuildings() []BuildingView

	KnownResources(category ResourceCategory) []ResourceView
	TurnNumber() int
	MapSize() int
	NumPlayers() int
	AllSpaces() []SpaceInfo
	AllRegions() []RegionInfo
	AvailableMercenaries() []Producible
	AllZones() []ZoneInfo
	// every player in the game in id order, and the player this view belongs to
	PlayerIDs() []int
	PlayerID() int
	PlayerScore(player_id int) int
	PlayerEliminated(player_id int) bool
	MatchingSpaces(condition SpaceCondition) []Coordinate
	// space-to-space steps over the board graph, so seams and links count as one step
	SpaceDistance(a Coordinate, b Coordinate) int
	SpaceNeighbors(space Coordinate) []Coordinate
	// SpaceDistance scaled to zone steps across spaces; zone offset distance within one space
	LocationDistance(a ZoneRef, b ZoneRef) int
	TechResearched(tech_id uint32) bool
	BuildingAvailable(building_id uint32) bool
	ValidBuildZone(building_id uint32, target ZoneRef) bool
	BuildCost(building_id uint32) Cost
	UnitCost(unit_id uint32) Cost
	TechCost(tech_id uint32) Cost
	ConfigValue(table string, field string, id uint32) (float64, error)

	MoveOrder(u UnitView, target ZoneRef, clear_previous bool) Order
	MoveSpaceOrder(u UnitView, target Coordinate, clear_previous bool) Order
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
	HireMercenaryOrder(unit_id uint32, target ZoneRef) Order
	CancelOrder(order_id uint64) Order
}
