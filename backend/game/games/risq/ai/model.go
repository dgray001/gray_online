package ai

import (
	"fmt"
	"os"
)

type Model interface {
	ApplyUpdate(view View, update_kind string)
	DecideOrders(view View) Decision
}

type Action interface {
	ToOrders(view View, internals *Internals) []Order
}

type Condition interface {
	Evaluate(view View, internals *Internals) bool
}

type NoopModel struct{}

func (NoopModel) ApplyUpdate(View, string)   {}
func (NoopModel) DecideOrders(View) Decision { return Decision{} }

// ParseModel parses an ai config blob into a Model, falling back to NoopModel on any error.
func ParseModel(raw map[string]any) Model {
	rules_raw, ok := raw["rules"].([]any)
	if !ok {
		fmt.Fprintln(os.Stderr, "ai config missing \"rules\" list")
		return NoopModel{}
	}
	rules, err := parseRules(rules_raw)
	if err != nil {
		fmt.Fprintln(os.Stderr, "ai config:", err)
		return NoopModel{}
	}
	depth, err := parseMaxBucketDepth(raw)
	if err != nil {
		fmt.Fprintln(os.Stderr, "ai config:", err)
		return NoopModel{}
	}
	return &RulesModel{rules: rules, internals: Internals{max_bucket_depth: depth}}
}

func parseMaxBucketDepth(raw map[string]any) (int, error) {
	value, present := raw["max_bucket_depth"]
	if !present {
		return defaultMaxBucketDepth, nil
	}
	depth, ok := value.(float64)
	if !ok || depth < 1 || depth != float64(int(depth)) {
		return 0, fmt.Errorf("\"max_bucket_depth\" must be a whole number of at least 1")
	}
	return int(depth), nil
}
