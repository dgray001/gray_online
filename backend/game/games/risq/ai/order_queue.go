package ai

const (
	QueueMoveSpace uint8 = iota
	QueueMoveZone
	QueueGather
	QueueBuild
	QueueRepair
	QueueRenew
	QueueAttackSpace
	QueueAttackZone
	QueueAttackUnit
	QueueAttackBuilding
	QueueAutoAttackUnit
	QueueAutoAttackBuilding
	QueueGarrison
	QueueUngarrison
	QueueDelete
	QueueCreate
	QueueResearch
)

type OrderEntry struct {
	SubjectUID                             uint64
	SubjectKind                            uint8
	Kind                                   uint8
	QueuePosition                          int
	TargetUID                              uint64
	HasTargetLocation                      bool
	HasTargetZone                          bool
	Target                                 ZoneRef
	ItemUnitID, ItemBuildingID, ItemTechID uint32
}

var QueueOrderNames = map[string]uint8{
	"move_space": QueueMoveSpace, "move_zone": QueueMoveZone, "gather": QueueGather, "build": QueueBuild,
	"repair": QueueRepair, "renew": QueueRenew, "attack_space": QueueAttackSpace, "attack_zone": QueueAttackZone,
	"attack_unit": QueueAttackUnit, "attack_building": QueueAttackBuilding, "auto_attack_unit": QueueAutoAttackUnit,
	"auto_attack_building": QueueAutoAttackBuilding, "garrison": QueueGarrison, "ungarrison": QueueUngarrison,
	"delete": QueueDelete, "create": QueueCreate, "research": QueueResearch,
}
