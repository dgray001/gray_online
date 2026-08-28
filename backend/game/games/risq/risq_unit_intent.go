package risq

import "fmt"

// Stamina spent per gather tick, capped to whatever stamina remains
const gatherTickStaminaCost = 3

// base_gather_speed is defined as resource gathered per this much stamina
const gatherRateStaminaBase = 10.0

type IntentKind interface {
	isIntentKind()
}

type RisqUnitIntent struct {
	intent_cost int
	min_cost    int
	max_cost    int
	detail      IntentKind
}

type MoveIntent struct {
	path      []*RisqZone
	next_step *RisqZone
	// true if the next step is a move within a space
	intra_step bool
}

func (*MoveIntent) isIntentKind() {}

type GatherIntent struct {
	resource *RisqResource
}

func (*GatherIntent) isIntentKind() {}

func createRisqUnitIntent() *RisqUnitIntent {
	return &RisqUnitIntent{}
}

func (i *RisqUnitIntent) resetIntent() {
	i.intent_cost = 0
	i.min_cost = 0
	i.max_cost = 0
	i.detail = nil
}

func (i *RisqUnitIntent) hasIntent() bool {
	return i.detail != nil
}

func (i *RisqUnitIntent) setMove(m *MoveIntent) {
	if m == nil {
		i.resetIntent()
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

func (i *RisqUnitIntent) setGather(resource *RisqResource) {
	i.detail = &GatherIntent{resource: resource}
	i.min_cost = 1
	i.max_cost = gatherTickStaminaCost
}

func (i *RisqUnitIntent) _printConsole(prefix string) {
	fmt.Println(prefix + "RisqUnitIntent {")
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
