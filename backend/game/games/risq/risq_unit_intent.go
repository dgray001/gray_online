package risq

type RisqUnitIntent struct {
	intent_cost int
	move        *MoveIntent
}

type MoveIntent struct {
	path      []*RisqZone
	next_step *RisqZone
	// true if the next step is a move within a space
	intra_step bool
}

func createRisqUnitIntent() *RisqUnitIntent {
	return &RisqUnitIntent{
		move: nil,
	}
}

func (i *RisqUnitIntent) resetIntent() {
	i.intent_cost = 0
	i.move = nil
}

func (i *RisqUnitIntent) hasIntent() bool {
	return i.move != nil
}

func (i *RisqUnitIntent) setMove(m *MoveIntent) {
	i.move = m
	if m != nil {
		if m.intra_step {
			i.intent_cost = 1
		} else {
			i.intent_cost = 6
		}
	}
}
