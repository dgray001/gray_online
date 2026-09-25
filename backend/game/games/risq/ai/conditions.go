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

type conditionBucketFull struct {
	bucket string
}

type conditionResourceAvailable struct {
	category ResourceCategory
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

func (c *conditionBucketFull) Evaluate(_ View, internals *Internals) bool {
	b := internals.Buckets[c.bucket]
	return b != nil && b.Desired > 0 && len(b.Members) >= b.Desired
}

func (c *conditionResourceAvailable) Evaluate(view View, _ *Internals) bool {
	home, ok := homeLocation(view)
	if !ok {
		return false
	}
	_, ok = view.NearestResource(home, c.category)
	return ok
}
