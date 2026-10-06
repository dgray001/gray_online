package risq

import "testing"

func TestMeetingStaminaSplit(t *testing.T) {
	cases := []struct {
		step_cost    uint
		stamina      int
		sunk, attack int
	}{
		{1, 15, 1, 2},
		{2, 15, 1, 2},
		{3, 15, 2, 1},
		{4, 15, 2, 1},
		{5, 15, 3, 0},
		{6, 15, 3, 0},
		{9, 15, 5, 0},
		{12, 15, 6, 0},
		{1, 2, 1, 1},
		{3, 3, 2, 1},
		{1, 1, 1, 0},
	}
	for _, c := range cases {
		sunk := halfMoveStamina(c.step_cost)
		if attack := meetingAttackStamina(sunk, c.stamina); sunk != c.sunk || attack != c.attack {
			t.Errorf("step %d with %d stamina: sunk %d attack %d, want sunk %d attack %d", c.step_cost, c.stamina, sunk, attack, c.sunk, c.attack)
		}
	}
}

func moverWith(step *RisqZone, cost uint) *RisqUnit {
	u := &RisqUnit{orderableBase: orderableBase{intent: createRisqIntent()}}
	u.intent.setMove(&MoveIntent{next_step: step, cost: cost})
	return u
}

func TestRestoreHalfMove(t *testing.T) {
	y, other := &RisqZone{}, &RisqZone{}

	u := moverWith(y, 6)
	u.restoreHalfMove(&halfMove{zone: y, paid: 3})
	if u.half_move == nil || u.intent.min_cost != 3 || u.intent.max_cost != 3 || u.intent.detail.(*MoveIntent).cost != 3 {
		t.Errorf("step into the cached zone: cache %v cost %d, want it kept with only the remaining 3 charged", u.half_move, u.intent.max_cost)
	}

	for name, u := range map[string]*RisqUnit{"a step into another zone": moverWith(other, 6), "no step at all": {orderableBase: orderableBase{intent: createRisqIntent()}}} {
		u.restoreHalfMove(&halfMove{zone: y, paid: 3})
		if u.half_move != nil || u.intent.max_cost != u.intent.min_cost {
			t.Errorf("%s: cache %v, want it dropped with no discount", name, u.half_move)
		}
	}
	elsewhere := moverWith(other, 6)
	elsewhere.restoreHalfMove(&halfMove{zone: y, paid: 3})
	if elsewhere.intent.max_cost != 6 {
		t.Errorf("step into another zone costs %d, want the full 6", elsewhere.intent.max_cost)
	}

	free := moverWith(y, 1)
	free.restoreHalfMove(&halfMove{zone: y, paid: 1})
	if free.intent.max_cost != 0 {
		t.Errorf("a 1-stamina step already half paid costs %d, want 0", free.intent.max_cost)
	}
}
