package risq

// Fields and methods RisqUnit and RisqBuilding both need to satisfy Orderable/Attackable, embedded
// rather than duplicated per type. Field/method promotion through the embed means every existing
// call site (u.deleted, u.isDeleted(), etc.) keeps working unchanged.
type orderableBase struct {
	deleted           bool
	internal_id       uint64
	player_id         int
	zone              *RisqZone
	turn_stamina      int
	current_stamina   int
	cs                RisqCombatStats
	order_queue       RisqOrderQueue
	intent            *RisqIntent
	attacked_by       []RisqDamageEvent
	interrupt_current bool
	target_priority   []TargetCategory
	attack_range      RisqRange
}

func (o *orderableBase) isDeleted() bool {
	return o.deleted
}

func (o *orderableBase) internalId() uint64 {
	return o.internal_id
}

func (o *orderableBase) playerId() int {
	return o.player_id
}

func (o *orderableBase) isAlive() bool {
	return o.cs.health > 0
}

func (o *orderableBase) applyDamage(event RisqDamageEvent) {
	o.cs.queueHealth(-event.damage)
	o.attacked_by = append(o.attacked_by, event)
}

func (o *orderableBase) activeOrders() []*RisqOrder {
	return o.order_queue.active_orders
}

func (o *orderableBase) refreshStamina() {
	o.current_stamina += o.turn_stamina
	max_stamina := maxStaminaFor(o.turn_stamina)
	if o.current_stamina > max_stamina {
		o.current_stamina = max_stamina
	}
	o.attacked_by = nil
}

func (o *orderableBase) inAttackRange(target *RisqZone) bool {
	return rangeCovers(o.zone, target, o.attack_range)
}

func (o *orderableBase) currentZone() *RisqZone {
	return o.zone
}
