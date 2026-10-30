package risq

type halfMove struct {
	zone *RisqZone
	paid int
}

func (u *RisqUnit) restoreHalfMove(paid *halfMove) {
	move, ok := u.intent.detail.(*MoveIntent)
	if paid == nil || !ok || move.next_step != paid.zone {
		return
	}
	move.cost -= min(move.cost, uint(paid.paid))
	u.intent.min_cost, u.intent.max_cost = int(move.cost), int(move.cost)
	u.half_move = paid
}

func (r *GameRisq) resolveMoveIntents(orderables []Orderable) int {
	removed := 0
	for _, actor := range orderables {
		unit, ok := actor.(*RisqUnit)
		if !ok {
			continue
		}
		if move, moving := unit.intent.detail.(*MoveIntent); moving {
			unit.resolveMoveCost(move, r)
			if !unit.intent.hasIntent() {
				unit.half_move = nil
				removed++
			}
		}
	}
	return removed
}

func (u *RisqUnit) resolveMoveCost(move *MoveIntent, r *GameRisq) {
	move.entry = u.resolvedWallEntry(move, r.blocking_walls, r.defended_walls)
	move.cost = u.path_profile.stepCost(move.next_step.space, !move.intra_step) + move.entry.cost
	paid := u.half_move
	u.intent.setMove(move)
	u.restoreHalfMove(paid)
	if u.tick_action != nil {
		r.snapshotIntent(u.tick_action, u.intent)
	}
	u.intent.resolveCost(u.current_stamina)
}

func (r *GameRisq) rejectBlockedMoves(orderables []Orderable) int {
	removed := 0
	for _, actor := range orderables {
		if unit, ok := actor.(*RisqUnit); ok {
			if move, moving := unit.intent.detail.(*MoveIntent); moving && move.entry.stopsMove() {
				unit.blockTickAction("blocked_by_wall")
				unit.intent.resetIntent()
				unit.half_move = nil
				removed++
			}
		}
	}
	return removed
}
