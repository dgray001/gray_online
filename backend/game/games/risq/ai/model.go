package ai

type Model interface {
	ApplyUpdate(view View, update_kind string)
	DecideOrders(view View) []Order
}

type Action interface {
	ToOrders(view View) []Order
}

type Condition interface {
	Evaluate(view View) bool
}

type NoopModel struct{}

func (NoopModel) ApplyUpdate(View, string)  {}
func (NoopModel) DecideOrders(View) []Order { return nil }

// ParseModel parses an ai config blob into a Model, falling back to NoopModel on any error.
func ParseModel(raw map[string]interface{}) Model {
	rules_raw, ok := raw["rules"].([]interface{})
	if !ok {
		return NoopModel{}
	}
	rules, err := parseRules(rules_raw)
	if err != nil {
		return NoopModel{}
	}
	return &RulesModel{rules: rules}
}
