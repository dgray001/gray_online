package ai

type conditionAll struct {
	conditions []Condition
}

type conditionAny struct {
	conditions []Condition
}

type conditionNot struct {
	condition Condition
}

type conditionAlways struct{}

type conditionTechResearched struct {
	tech_id uint32
}

func evaluateCondition(condition Condition, view View, internals *Internals) (bool, bool) {
	previous := internals.eval_errors
	result := condition.Evaluate(view, internals)
	return result, internals.eval_errors == previous
}

func (c *conditionAll) Evaluate(view View, internals *Internals) bool {
	for _, condition := range c.conditions {
		if !condition.Evaluate(view, internals) {
			return false
		}
	}
	return true
}

func (c *conditionAny) Evaluate(view View, internals *Internals) bool {
	for _, condition := range c.conditions {
		if condition.Evaluate(view, internals) {
			return true
		}
	}
	return false
}

func (c *conditionNot) Evaluate(view View, internals *Internals) bool {
	return !c.condition.Evaluate(view, internals)
}

func (c *conditionAlways) Evaluate(View, *Internals) bool {
	return true
}

func (c *conditionTechResearched) Evaluate(view View, _ *Internals) bool {
	return view.TechResearched(c.tech_id)
}
