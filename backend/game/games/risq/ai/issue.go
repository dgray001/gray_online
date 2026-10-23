package ai

type Verb uint8

const (
	VerbMoveSpace Verb = iota
	VerbMoveZone
	VerbGather
	VerbBuild
	VerbRepair
	VerbRenew
	VerbAttackUnit
	VerbAttackBuilding
	VerbAttackSpace
	VerbAttackZone
	VerbGarrison
	VerbUngarrison
	VerbDeleteUnit
	VerbCreateUnit
	VerbResearch
	VerbDeleteBuilding
	VerbBuildingAttackUnit
	VerbBuildingAttackBuilding
	VerbCancelFoundation
	VerbHire
	VerbCancelOrder
)

type IssueTarget struct {
	Unit     *UnitView
	Building *BuildingView
	Resource *ResourceView
	Space    Coordinate
	Zone     ZoneRef
}

type IssueRequest struct {
	Verb       Verb
	Unit       UnitView
	Building   BuildingView
	Foundation FoundationView
	Target     IssueTarget
	Item       uint32
	OrderID    uint64
	Clear      bool
}

func DispatchIssue(v View, r IssueRequest) Order {
	t := r.Target
	switch r.Verb {
	case VerbMoveSpace:
		return v.MoveSpaceOrder(r.Unit, t.Space, r.Clear)
	case VerbMoveZone:
		return v.MoveOrder(r.Unit, t.Zone, r.Clear)
	case VerbGather:
		return v.GatherOrder(r.Unit, *t.Resource, r.Clear)
	case VerbBuild:
		return v.BuildOrder(r.Unit, r.Item, t.Zone, r.Clear)
	case VerbRepair:
		return v.RepairOrder(r.Unit, *t.Building, r.Clear)
	case VerbRenew:
		return v.RenewOrder(r.Unit, *t.Building, r.Clear)
	case VerbAttackUnit:
		return v.AttackUnitOrder(r.Unit, *t.Unit, r.Clear)
	case VerbAttackBuilding:
		return v.AttackBuildingOrder(r.Unit, *t.Building, r.Clear)
	case VerbAttackSpace:
		return v.AttackSpaceOrder(r.Unit, t.Space, r.Clear)
	case VerbAttackZone:
		return v.AttackZoneOrder(r.Unit, t.Zone, r.Clear)
	case VerbGarrison:
		return v.GarrisonOrder(r.Unit, *t.Building, r.Clear)
	case VerbUngarrison:
		return v.UngarrisonOrder(r.Unit, r.Clear)
	case VerbDeleteUnit:
		return v.DeleteUnitOrder(r.Unit)
	case VerbCreateUnit:
		return v.CreateUnitOrder(r.Building, r.Item)
	case VerbResearch:
		return v.ResearchOrder(r.Building, r.Item)
	case VerbDeleteBuilding:
		return v.DeleteBuildingOrder(r.Building)
	case VerbBuildingAttackUnit:
		return v.BuildingAttackUnitOrder(r.Building, *t.Unit)
	case VerbBuildingAttackBuilding:
		return v.BuildingAttackBuildingOrder(r.Building, *t.Building)
	case VerbCancelFoundation:
		return v.CancelFoundationOrder(r.Foundation)
	case VerbHire:
		return v.HireMercenaryOrder(r.Item, t.Zone)
	case VerbCancelOrder:
		return v.CancelOrder(r.OrderID)
	}
	return Order{}
}
