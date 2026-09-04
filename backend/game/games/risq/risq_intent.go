package risq

import "math"

func maxStaminaFor(turn_stamina int) int {
	return int(math.Ceil(float64(turn_stamina) * 1.5))
}

type IntentKind interface {
	isIntentKind()
}

type RisqIntent struct {
	intent_cost int
	min_cost    int
	max_cost    int
	detail      IntentKind
}

func createRisqIntent() *RisqIntent {
	return &RisqIntent{}
}

func (i *RisqIntent) resetIntent() {
	i.intent_cost = 0
	i.min_cost = 0
	i.max_cost = 0
	i.detail = nil
}

func (i *RisqIntent) hasIntent() bool {
	return i.detail != nil
}

type DeleteIntent struct{}

func (*DeleteIntent) isIntentKind() {}

func (i *RisqIntent) setDelete() {
	i.detail = &DeleteIntent{}
	i.min_cost = 0
	i.max_cost = 0
}

func (i *RisqIntent) resolveCost(current_stamina int) {
	if !i.hasIntent() {
		return
	}
	if current_stamina < i.min_cost {
		i.resetIntent()
	} else if current_stamina < i.max_cost {
		i.intent_cost = current_stamina
	} else {
		i.intent_cost = i.max_cost
	}
}
