package risq

import "fmt"

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

func (i *RisqUnitIntent) _printConsole(prefix string) {
	fmt.Println(prefix + "RisqUnitIntent {")
	fmt.Println(prefix+"  intent_cost:", i.intent_cost)
	fmt.Print(prefix + "  move:")
	i.move.printConsole("  ")
	fmt.Println(prefix + "}")
}

func (m *MoveIntent) printConsole(prefix string) {
	fmt.Println(prefix + "MoveIntent {")
	fmt.Println(prefix + "  path:")
	fmt.Println(prefix + "  next_step:")
	fmt.Println(prefix+"  intra_step:", m.intra_step)
	fmt.Println(prefix + "}")
}
