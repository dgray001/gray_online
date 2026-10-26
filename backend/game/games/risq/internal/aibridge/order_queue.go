package aibridge

import (
	"github.com/dgray001/gray_online/game/games/risq/ai"
	"github.com/dgray001/gray_online/game/games/risq/internal/defs"
	"github.com/dgray001/gray_online/util"
	"maps"
	"slices"
)

func (v *aiView) initQueues() {
	if v.queues != nil {
		return
	}
	v.queues = map[uint64][]snapOrder{}
	for _, u := range v.me.Units {
		v.queues[u.InternalId] = slices.Clone(u.ActiveOrders)
	}
	for _, b := range v.me.Buildings {
		v.queues[1_000_000_000+b.InternalId] = slices.Clone(b.ActiveOrders)
	}
	v.queue_subjects = map[uint64][]uint64{}
	for subject, queue := range v.queues {
		for _, order := range queue {
			v.queue_subjects[order.InternalId] = append(v.queue_subjects[order.InternalId], subject)
		}
	}
}

func (v *aiView) recordOrder(subject uint64, order defs.OrderType, target int64, clear bool) {
	v.initQueues()
	queue := v.queues[subject]
	if clear {
		queue = slices.DeleteFunc(queue, func(o snapOrder) bool { return !o.OrderType.IsClearImmune() })
	}
	v.queues[subject] = append(queue, snapOrder{OrderType: order, TargetId: target})
	v.revision++
	v.refreshSubject(subject)
}

func (v *aiView) refreshSubject(subject uint64) {
	if subject < 1_000_000_000 {
		v.refreshUnit(subject)
	}
	v.refreshIdle(subject)
}

func (v *aiView) cancelQueueOrder(id uint64) []uint64 {
	v.initQueues()
	subjects := v.queue_subjects[id]
	for _, subject := range subjects {
		v.queues[subject] = slices.DeleteFunc(v.queues[subject], func(o snapOrder) bool { return o.InternalId == id })
	}
	delete(v.queue_subjects, id)
	return subjects
}

func queueKind(kind defs.OrderType) uint8 {
	switch kind {
	case defs.OrderType_BuildingDelete:
		return ai.QueueOrderNames["delete"]
	case defs.OrderType_BuildingAttackUnit:
		return ai.QueueOrderNames["attack_unit"]
	case defs.OrderType_BuildingAttackBuilding:
		return ai.QueueOrderNames["attack_building"]
	case defs.OrderType_BuildingAutoAttackUnit:
		return ai.QueueOrderNames["auto_attack_unit"]
	case defs.OrderType_BuildingAutoAttackBuilding:
		return ai.QueueOrderNames["auto_attack_building"]
	}
	return uint8(kind - defs.OrderType_UnitMoveSpace)
}

func (v *aiView) queueEntry(subject uint64, position int, raw snapOrder) ai.OrderEntry {
	e := ai.OrderEntry{SubjectUID: subject, Kind: queueKind(raw.OrderType), QueuePosition: position}
	if raw.OrderType.IsBuildingOrder() {
		e.SubjectKind = 1
	}
	v.queueTarget(&e, raw)
	return e
}

func (v *aiView) queueTarget(e *ai.OrderEntry, raw snapOrder) {
	switch e.Kind {
	case ai.QueueAttackUnit, ai.QueueAutoAttackUnit:
		e.TargetUID = uint64(raw.TargetId)
	case ai.QueueRepair, ai.QueueRenew, ai.QueueAttackBuilding, ai.QueueAutoAttackBuilding, ai.QueueGarrison:
		e.TargetUID = 1_000_000_000 + uint64(raw.TargetId)
	case ai.QueueCreate:
		e.ItemUnitID = uint32(raw.TargetId)
	case ai.QueueResearch:
		e.ItemTechID = uint32(raw.TargetId)
	case ai.QueueBuild:
		item, zone := util.InvertPair(uint(raw.TargetId))
		e.ItemBuildingID = uint32(item)
		e.Target, e.HasTargetLocation, e.HasTargetZone = zoneRefFromKey(uint(zone)), true, true
	}
	if e.TargetUID != 0 {
		v.queueEntityLocation(e)
	}
	if e.Kind == ai.QueueBuild {
		e.TargetUID = ai.FoundationUID(e.Target, v.playerId())
	}
	if e.Kind == ai.QueueMoveSpace || e.Kind == ai.QueueAttackSpace {
		x, y := util.InvertPair(uint(raw.TargetId))
		e.Target.Space, e.HasTargetLocation = ai.Coordinate{X: x, Y: y}, true
	}
	v.queueZoneTarget(e, raw)
}

func (v *aiView) queueEntityLocation(e *ai.OrderEntry) {
	if e.Kind == ai.QueueAttackUnit || e.Kind == ai.QueueAutoAttackUnit {
		if u := v.units[e.TargetUID]; u != nil {
			e.Target, e.HasTargetLocation, e.HasTargetZone = v.unitLocation(u), true, true
		}
	} else if b := v.buildings[e.TargetUID-1_000_000_000]; b != nil {
		e.Target, e.HasTargetLocation, e.HasTargetZone = zoneRefOf(b.Space, b.Zone), true, true
	}
}

func (v *aiView) queueZoneTarget(e *ai.OrderEntry, raw snapOrder) {
	if e.Kind != ai.QueueMoveZone && e.Kind != ai.QueueGather && e.Kind != ai.QueueAttackZone {
		return
	}
	e.Target, e.HasTargetLocation, e.HasTargetZone = zoneRefFromKey(uint(raw.TargetId)), true, true
	if e.Kind == ai.QueueGather {
		if r, ok := v.gatherTargetAt(e.Target); ok {
			e.TargetUID = 2_000_000_000 + r.InternalID
			if r.IsBuilding {
				e.TargetUID = 1_000_000_000 + r.InternalID
			}
		}
	}
}

func (v *aiView) buildOrderQueue() []ai.OrderEntry {
	v.initQueues()
	entries := make([]ai.OrderEntry, 0)
	for _, subject := range slices.Sorted(maps.Keys(v.queues)) {
		for position, raw := range v.queues[subject] {
			entries = append(entries, v.queueEntry(subject, position, raw))
		}
	}
	return entries
}
