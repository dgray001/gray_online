package risq

// Damage stance-picked attackers have committed against each target this tick, so later attackers pick a different target instead of overkilling
func (r *GameRisq) projectDamage(attacker *RisqUnit, target Attackable) {
	if r.projected_damage == nil {
		r.projected_damage = make(map[Attackable]float64)
	}
	attacker_cs := attacker.combatStats(r, target, true)
	target_cs := target.combatStats(r, attacker, false)
	r.projected_damage[target] += combatDamage(&attacker_cs, &target_cs, min(attacker.current_stamina, unitTickStaminaCost))
}

func (r *GameRisq) projectedDead(target Attackable) bool {
	committed, ok := r.projected_damage[target]
	if !ok {
		return false
	}
	switch t := target.(type) {
	case *RisqUnit:
		return committed >= t.cs.health
	case *RisqBuilding:
		return committed >= t.cs.health
	}
	return false
}
