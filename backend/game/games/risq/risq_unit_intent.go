package risq

const unitTickStaminaCost = 3

const gatherRateStaminaBase = 10.0

const repairSpeedFactor = 0.6
const repairCostFactor = 1.0

type MoveIntent struct {
	path       []*RisqZone
	next_step  *RisqZone
	intra_step bool
}

func (*MoveIntent) isIntentKind() {}

type GatherIntent struct {
	resource *RisqResource
}

func (*GatherIntent) isIntentKind() {}

func (i *RisqIntent) setMove(m *MoveIntent) {
	if m == nil {
		i.detail = nil
		i.min_cost = 0
		i.max_cost = 0
		return
	}
	i.detail = m
	if m.intra_step {
		i.min_cost = 1
		i.max_cost = 1
	} else {
		i.min_cost = 6
		i.max_cost = 6
	}
}

func (i *RisqIntent) setGather(resource *RisqResource) {
	i.detail = &GatherIntent{resource: resource}
	i.min_cost = 1
	i.max_cost = unitTickStaminaCost
}

type AttackBuildingIntent struct {
	target *RisqBuilding
}

func (*AttackBuildingIntent) isIntentKind() {}

func (i *RisqIntent) setAttackBuilding(target *RisqBuilding) {
	i.detail = &AttackBuildingIntent{target: target}
	i.min_cost = 1
	i.max_cost = unitTickStaminaCost
}

type RepairIntent struct {
	target *RisqBuilding
}

func (*RepairIntent) isIntentKind() {}

func (i *RisqIntent) setRepair(target *RisqBuilding) {
	i.detail = &RepairIntent{target: target}
	i.min_cost = 1
	i.max_cost = unitTickStaminaCost
}

type ConstructionIntent struct {
	building_under_construction *RisqBuilding
	building_id                 uint32
	zone                        *RisqZone
}

func (*ConstructionIntent) isIntentKind() {}

func (i *RisqIntent) setBuild(building_under_construction *RisqBuilding, building_id uint32, zone *RisqZone) {
	i.detail = &ConstructionIntent{building_under_construction: building_under_construction, building_id: building_id, zone: zone}
	i.min_cost = 1
	i.max_cost = unitTickStaminaCost
}
