package risq

import (
	"math"
	"sort"

	"github.com/dgray001/gray_online/util"
)

const unitTickStaminaCost = 3

const gatherRoundingPlaces = 4

const gatherRateStaminaBase = 10.0

const repairSpeedFactor = 0.6
const repairCostFactor = 1.0

// Returns the heal amount and resource cost for spending stamina repairing building; ok is false if unrepairable
func repairHealAndCost(building *RisqBuilding, stamina int) (heal float64, cost RisqResourceCost, ok bool) {
	config := buildingConfigs[building.building_id]
	if config.build_stamina <= 0 {
		return 0, RisqResourceCost{}, false
	}
	max_health := float64(building.cs.max_health)
	heal = repairSpeedFactor * max_health / float64(config.build_stamina) * float64(stamina)
	if remaining := max_health - building.cs.health; heal > remaining {
		heal = remaining
	}
	cost = config.cost.scale(repairCostFactor * heal / max_health)
	return heal, cost, true
}

type MoveIntent struct {
	path       []*RisqZone
	next_step  *RisqZone
	intra_step bool
}

func (*MoveIntent) isIntentKind() {}

type GatherIntent struct {
	source Gatherable
}

func (*GatherIntent) isIntentKind() {}

type gatherDemand struct {
	unit   *RisqUnit
	amount float64
}

// Computes each gatherer's actual allotment for a contested resource: whoever asks for less than an
// equal share gets their full request, and only the leftover is split among those who asked for more
func computeGatherAllotments(orderables []Orderable) map[*RisqUnit]float64 {
	by_source := make(map[Gatherable][]gatherDemand)
	for _, o := range orderables {
		u, ok := o.(*RisqUnit)
		if !ok || !u.intent.hasIntent() {
			continue
		}
		gather, ok := u.intent.detail.(*GatherIntent)
		if !ok {
			continue
		}
		amount := float64(u.intent.intent_cost) * (float64(gather.source.gatherSpeed()) / gatherRateStaminaBase)
		by_source[gather.source] = append(by_source[gather.source], gatherDemand{unit: u, amount: amount})
	}
	allotments := make(map[*RisqUnit]float64)
	for source, demands := range by_source {
		for unit, amount := range quantizeAllotments(waterFillGather(demands, source.gatherResourcesLeft())) {
			allotments[unit] = amount
		}
	}
	return allotments
}

// Rounds each gatherer's allotment to gatherRoundingPlaces while conserving their exact sum, so the total
// deducted from a resource doesn't depend on which order gatherers happened to be processed in
func quantizeAllotments(raw map[*RisqUnit]float64) map[*RisqUnit]float64 {
	scale := math.Pow(10, gatherRoundingPlaces)
	total := 0.0
	for _, amount := range raw {
		total += amount
	}
	target_units := int64(math.Round(util.RoundTo(total, gatherRoundingPlaces) * scale))
	type floored struct {
		unit      *RisqUnit
		amount    float64
		remainder float64
	}
	floors := make([]floored, 0, len(raw))
	floor_units := int64(0)
	for unit, amount := range raw {
		scaled := amount * scale
		floor := math.Floor(scaled)
		floors = append(floors, floored{unit: unit, amount: floor / scale, remainder: scaled - floor})
		floor_units += int64(floor)
	}
	sort.Slice(floors, func(i, j int) bool {
		if floors[i].remainder != floors[j].remainder {
			return floors[i].remainder > floors[j].remainder
		}
		return floors[i].unit.internal_id < floors[j].unit.internal_id
	})
	result := make(map[*RisqUnit]float64, len(floors))
	leftover := target_units - floor_units
	for i, f := range floors {
		amount := f.amount
		if int64(i) < leftover {
			amount += 1 / scale
		}
		result[f.unit] = amount
	}
	return result
}

func waterFillGather(demands []gatherDemand, available float64) map[*RisqUnit]float64 {
	result := make(map[*RisqUnit]float64, len(demands))
	remaining := demands
	for len(remaining) > 0 {
		fair_share := available / float64(len(remaining))
		next := remaining[:0]
		progressed := false
		for _, d := range remaining {
			if d.amount <= fair_share {
				result[d.unit] = d.amount
				available -= d.amount
				progressed = true
			} else {
				next = append(next, d)
			}
		}
		remaining = next
		if !progressed {
			fair_share = available / float64(len(remaining))
			for _, d := range remaining {
				result[d.unit] = fair_share
			}
			break
		}
	}
	return result
}

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

func (i *RisqIntent) setGather(source Gatherable) {
	i.detail = &GatherIntent{source: source}
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

type AttackUnitIntent struct {
	target *RisqUnit
}

func (*AttackUnitIntent) isIntentKind() {}

func (i *RisqIntent) setAttackUnit(target *RisqUnit) {
	i.detail = &AttackUnitIntent{target: target}
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

type GarrisonIntent struct {
	target *RisqBuilding
}

func (*GarrisonIntent) isIntentKind() {}

func (i *RisqIntent) setGarrison(target *RisqBuilding) {
	i.detail = &GarrisonIntent{target: target}
	i.min_cost = 1
	i.max_cost = 1
}

// Deterministically settle same-tick garrison attempts
func computeGarrisonAllotments(orderables []Orderable) map[*RisqUnit]bool {
	by_building := make(map[*RisqBuilding][]*RisqUnit)
	for _, o := range orderables {
		u, ok := o.(*RisqUnit)
		if !ok || !u.intent.hasIntent() {
			continue
		}
		garrison, ok := u.intent.detail.(*GarrisonIntent)
		if !ok {
			continue
		}
		by_building[garrison.target] = append(by_building[garrison.target], u)
	}
	allotted := make(map[*RisqUnit]bool)
	for building, units := range by_building {
		sort.Slice(units, func(i, j int) bool { return units[i].internal_id < units[j].internal_id })
		remaining := int(building.garrison_capacity) - len(building.garrisoned_units)
		for i, u := range units {
			if i >= remaining {
				break
			}
			allotted[u] = true
		}
	}
	return allotted
}

type UngarrisonIntent struct {
	next_step *RisqZone
}

func (*UngarrisonIntent) isIntentKind() {}

func (i *RisqIntent) setUngarrison(next_step *RisqZone) {
	i.detail = &UngarrisonIntent{next_step: next_step}
	i.min_cost = 1
	i.max_cost = 1
}
