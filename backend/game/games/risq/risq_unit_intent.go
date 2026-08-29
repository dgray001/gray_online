package risq

import "fmt"

\const gatherTickStaminaCost = 3

\const gatherRateStaminaBase = 10.0

type MoveIntent struct {
	path      []*RisqZone
	next_step *RisqZone
	intra_step bool
}

func (*MoveIntent) isIntentKind() {}

type GatherIntent struct {
	resource *RisqResource
}

func (*GatherIntent) isIntentKind() {}

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

func (i *RisqIntent) setGather(resource *RisqResource) {
	i.detail = &GatherIntent{resource: resource}
	i.min_cost = 1
	i.max_cost = gatherTickStaminaCost
}

func (i *RisqIntent) _printConsole(prefix string) {
	fmt.Println(prefix + "RisqIntent {")
	fmt.Println(prefix+"  intent_cost:", i.intent_cost)
	if move, ok := i.detail.(*MoveIntent); ok {
		fmt.Print(prefix + "  move:")
		move.printConsole("  ")
	}
	fmt.Println(prefix + "}")
}

func (m *MoveIntent) printConsole(prefix string) {
	fmt.Println(prefix + "MoveIntent {")
	fmt.Println(prefix + "  path:")
	fmt.Println(prefix + "  next_step:")
	fmt.Println(prefix+"  intra_step:", m.intra_step)
	fmt.Println(prefix + "}")
}
