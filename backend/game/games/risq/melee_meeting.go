package risq

import (
	"github.com/dgray001/gray_online/game/games/risq/internal/defs"
	"github.com/dgray001/gray_online/util"
)

// The half move a unit already paid toward the adjacent zone it keeps stepping into
type halfMove struct {
	zone *RisqZone
	paid int
}

// Keeps a paid half move while this tick's step goes into the same zone, charging only the rest of the step
func (u *RisqUnit) restoreHalfMove(paid *halfMove) {
	move, ok := u.intent.detail.(*MoveIntent)
	if paid == nil || !ok || move.next_step != paid.zone {
		return
	}
	move.cost -= min(move.cost, uint(paid.paid))
	u.intent.min_cost, u.intent.max_cost = int(move.cost), int(move.cost)
	u.half_move = paid
}

// Replaces u's step with the half move (paid once per turn) and the attack of a meeting at the border; u stays in its zone
func (u *RisqUnit) meetAtBorder(target Attackable, move *MoveIntent, tick uint16, turn uint16) {
	sunk, attack := 0, min(unitTickStaminaCost, u.current_stamina)
	if u.half_move == nil {
		sunk = halfMoveStamina(move.cost)
		attack = meetingAttackStamina(sunk, u.current_stamina)
		u.half_move = &halfMove{zone: move.next_step, paid: sunk}
	}
	if sunk+attack == 0 {
		u.intent.resetIntent()
		u.half_move, u.move_path = nil, nil
		return
	}
	u.intent.detail = &UnitAttackIntent{target: target}
	u.intent.sunk_cost, u.intent.intent_cost = sunk, sunk+attack
	u.move_path = nil
}

// Decides every meeting from the intents as they stand, then applies them, so no unit's change can affect another's decision
// Returns how many units were left with nothing to spend and so no intent
func (r *GameRisq) resolveMeleeMeetings(orderables []Orderable) int {
	type meeting struct {
		unit   *RisqUnit
		target Attackable
		move   *MoveIntent
	}
	meetings := make([]meeting, 0)
	for _, o := range orderables {
		if u, ok := o.(*RisqUnit); ok {
			if target, meets := u.meetingTarget(r.blocking_walls); meets {
				meetings = append(meetings, meeting{u, target, u.intent.detail.(*MoveIntent)})
			}
		}
	}
	removed := 0
	for _, m := range meetings {
		m.unit.meetAtBorder(m.target, m.move, r.current_tick+1, r.turn_number)
		if !m.unit.intent.hasIntent() {
			removed++
		} else {
			util.Debugf(r.std, "Meeting at border: unit %d meets %d tick=%d turn=%d sunk=%d attack=%d", m.unit.internal_id, m.target.internalId(), r.current_tick+1, r.turn_number, m.unit.intent.sunk_cost, m.unit.intent.intent_cost-m.unit.intent.sunk_cost)
		}
	}
	return removed
}

// A chased or interrupt-eligible enemy stepping across the same border in the opposite direction
func (u *RisqUnit) meetingTarget(walls map[*RisqBuilding]bool) (Attackable, bool) {
	move, ok := u.intent.detail.(*MoveIntent)
	if !ok || u.zone == nil {
		return nil, false
	}
	if radius, ranged := u.attack_range.SpaceRadius(); ranged && (radius > 0 || move.intra_step) {
		return nil, false
	}
	if wall := move.next_step.building; move.attacking && wall != nil && move.next_step.wallBlocksEntry(u.zone, u.player_id, walls) {
		return wall, true
	}
	target := move.chasing
	if target == nil {
		target = u.crossingAttackTarget(move)
	}
	if target == nil || !target.isAlive() || target.zone == nil || move.next_step != target.zone {
		return nil, false
	}
	target_move, ok := target.intent.detail.(*MoveIntent)
	return target, ok && target_move.next_step == u.zone
}

func (u *RisqUnit) crossingAttackTarget(move *MoveIntent) *RisqUnit {
	if !u.interrupt_current || u.stance == defs.UnitStance_PASSIVE || u.unitType() == defs.UnitType_ECONOMIC || u.cs.attack_type == defs.AttackType_NONE || u.holdsAssignedAttack() {
		return nil
	}
	if move.next_step.space.getVisibility(u.player_id) < defs.VisibilityGood {
		return nil
	}
	best := newCategoryBest()
	for _, target := range move.next_step.units {
		target_move, moving := target.intent.detail.(*MoveIntent)
		if !moving || target_move.next_step != u.zone || target.deleted || !target.isAlive() || !canAttack(u.player_id, target.player_id) || !targetInPriority(target, u.target_priority) {
			continue
		}
		best.considerUnit(target, 0)
	}
	target, _ := best.pick(u.target_priority).(*RisqUnit)
	return target
}

// Stamina a melee unit pays to walk half a step to the border before meeting a unit crossing it
func halfMoveStamina(step_cost uint) int {
	return int((step_cost + 1) / 2)
}

// Attack stamina left in a tick after the half move, never more than the unit has
func meetingAttackStamina(sunk int, current_stamina int) int {
	return max(0, min(unitTickStaminaCost-sunk, current_stamina-sunk))
}
