package risq

import "github.com/dgray001/gray_online/util"

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
func (u *RisqUnit) meetAtBorder(target *RisqUnit, move *MoveIntent, tick uint16, turn uint16) {
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
	util.DebugLog.Printf("Meeting at border: unit %d meets %d tick=%d turn=%d sunk=%d attack=%d", u.internal_id, target.internal_id, tick, turn, sunk, attack)
}

// Decides every meeting from the intents as they stand, then applies them, so no unit's change can affect another's decision
// Returns how many units were left with nothing to spend and so no intent
func (r *GameRisq) resolveMeleeMeetings(orderables []Orderable) int {
	type meeting struct {
		unit, target *RisqUnit
		move         *MoveIntent
	}
	meetings := make([]meeting, 0)
	for _, o := range orderables {
		if u, ok := o.(*RisqUnit); ok {
			if target, meets := u.meetingTarget(); meets {
				meetings = append(meetings, meeting{u, target, u.intent.detail.(*MoveIntent)})
			}
		}
	}
	removed := 0
	for _, m := range meetings {
		m.unit.meetAtBorder(m.target, m.move, r.current_tick+1, r.turn_number)
		if !m.unit.intent.hasIntent() {
			removed++
		}
	}
	return removed
}

// The unit u is closing on, when that unit is in the adjacent zone u steps into and is itself stepping into u's zone
func (u *RisqUnit) meetingTarget() (*RisqUnit, bool) {
	move, ok := u.intent.detail.(*MoveIntent)
	if !ok || move.chasing == nil || u.zone == nil {
		return nil, false
	}
	if radius, ranged := u.attack_range.SpaceRadius(); ranged && (radius > 0 || move.intra_step) {
		return nil, false
	}
	target := move.chasing
	if !target.isAlive() || target.zone == nil || move.next_step != target.zone {
		return nil, false
	}
	target_move, ok := target.intent.detail.(*MoveIntent)
	return target, ok && target_move.next_step == u.zone
}

// Stamina a melee unit pays to walk half a step to the border before meeting a unit crossing it
func halfMoveStamina(step_cost uint) int {
	return int((step_cost + 1) / 2)
}

// Attack stamina left in a tick after the half move, never more than the unit has
func meetingAttackStamina(sunk int, current_stamina int) int {
	return max(0, min(unitTickStaminaCost-sunk, current_stamina-sunk))
}
