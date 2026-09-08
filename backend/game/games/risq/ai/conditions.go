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

type conditionBuildingCountAtLeast struct {
	building_id *uint32
	count       int
}

type conditionBuildingCountAtMost struct {
	building_id *uint32
	count       int
}

type conditionPopulationHeadroomAtLeast struct {
	amount int
}

type conditionPopulationHeadroomAtMost struct {
	amount int
}

type conditionPopulationAtLeast struct {
	amount  int
	unit_id *uint32
}

type conditionPopulationAtMost struct {
	amount  int
	unit_id *uint32
}

func (c *conditionAll) Evaluate(view View) bool {
	for _, condition := range c.conditions {
		if !condition.Evaluate(view) {
			return false
		}
	}
	return true
}

func (c *conditionAny) Evaluate(view View) bool {
	for _, condition := range c.conditions {
		if condition.Evaluate(view) {
			return true
		}
	}
	return false
}

func (c *conditionNot) Evaluate(view View) bool {
	return !c.condition.Evaluate(view)
}

func (c *conditionAlways) Evaluate(View) bool {
	return true
}

func (c *conditionBuildingCountAtLeast) Evaluate(view View) bool {
	if c.building_id == nil {
		return len(view.Buildings()) >= c.count
	}
	n := 0
	for _, b := range view.Buildings() {
		if b.BuildingID == *c.building_id {
			n++
		}
	}
	return n >= c.count
}

func (c *conditionBuildingCountAtMost) Evaluate(view View) bool {
	if c.building_id == nil {
		return len(view.Buildings()) <= c.count
	}
	n := 0
	for _, b := range view.Buildings() {
		if b.BuildingID == *c.building_id {
			n++
		}
	}
	return n <= c.count
}

func (c *conditionPopulationHeadroomAtLeast) Evaluate(view View) bool {
	current, limit := view.Population()
	return limit-current >= c.amount
}

func (c *conditionPopulationHeadroomAtMost) Evaluate(view View) bool {
	current, limit := view.Population()
	return limit-current <= c.amount
}

func (c *conditionPopulationAtLeast) Evaluate(view View) bool {
	if c.unit_id == nil {
		current, _ := view.Population()
		return current >= c.amount
	}
	count := 0
	for _, u := range view.Units() {
		if u.UnitID == *c.unit_id {
			count++
		}
	}
	return count >= c.amount
}

func (c *conditionPopulationAtMost) Evaluate(view View) bool {
	if c.unit_id == nil {
		current, _ := view.Population()
		return current <= c.amount
	}
	count := 0
	for _, u := range view.Units() {
		if u.UnitID == *c.unit_id {
			count++
		}
	}
	return count <= c.amount
}
