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

type conditionBuildingCountEquals struct {
	building_id *uint32
	count       int
}

type conditionPopulationHeadroomAtLeast struct {
	amount int
}

type conditionPopulationHeadroomAtMost struct {
	amount int
}

type conditionPopulationHeadroomEquals struct {
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

type conditionPopulationEquals struct {
	amount  int
	unit_id *uint32
}

func buildingCount(view View, building_id *uint32) int {
	if building_id == nil {
		return len(view.Buildings())
	}
	n := 0
	for _, b := range view.Buildings() {
		if b.BuildingID == *building_id {
			n++
		}
	}
	return n
}

func populationCount(view View, unit_id *uint32) int {
	if unit_id == nil {
		current, _ := view.Population()
		return current
	}
	n := 0
	for _, u := range view.Units() {
		if u.UnitID == *unit_id {
			n++
		}
	}
	return n
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
	return buildingCount(view, c.building_id) >= c.count
}

func (c *conditionBuildingCountAtMost) Evaluate(view View) bool {
	return buildingCount(view, c.building_id) <= c.count
}

func (c *conditionBuildingCountEquals) Evaluate(view View) bool {
	return buildingCount(view, c.building_id) == c.count
}

func (c *conditionPopulationHeadroomAtLeast) Evaluate(view View) bool {
	current, limit := view.Population()
	return limit-current >= c.amount
}

func (c *conditionPopulationHeadroomAtMost) Evaluate(view View) bool {
	current, limit := view.Population()
	return limit-current <= c.amount
}

func (c *conditionPopulationHeadroomEquals) Evaluate(view View) bool {
	current, limit := view.Population()
	return limit-current == c.amount
}

func (c *conditionPopulationAtLeast) Evaluate(view View) bool {
	return populationCount(view, c.unit_id) >= c.amount
}

func (c *conditionPopulationAtMost) Evaluate(view View) bool {
	return populationCount(view, c.unit_id) <= c.amount
}

func (c *conditionPopulationEquals) Evaluate(view View) bool {
	return populationCount(view, c.unit_id) == c.amount
}
