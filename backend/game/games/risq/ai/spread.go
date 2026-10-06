package ai

import "fmt"

type spreadOverflow uint8

const (
	spreadOverflowRoundRobin spreadOverflow = iota
	spreadOverflowNone
)

// How a picker shares its units over its candidates. Every expression sees the unit_*, target_* and assign_round variables.
type spreadConfig struct {
	capacity   *amount // attackers one candidate can absorb; unlimited when absent
	unit_score amount  // added to the picker's score for one unit and candidate
	unit_order amount  // units pick in descending order of this, ties by internal id
	weight     amount
	by_space   bool
	queue      amount // follow-up rounds after the first pick
	overflow   spreadOverflow
}

var spreadKeys = map[string]bool{"capacity": true, "unit_score": true, "unit_order": true, "weight": true, "group_by": true, "queue": true, "overflow": true}

func parseSpread(raw any) (*spreadConfig, error) {
	obj, ok := raw.(map[string]any)
	if !ok {
		return nil, fmt.Errorf("\"spread\" must be an object")
	}
	for key := range obj {
		if !spreadKeys[key] {
			return nil, fmt.Errorf("unknown spread option %q", key)
		}
	}
	s := &spreadConfig{}
	var err error
	if s.weight, err = parseNumber(obj, "weight", 1); err != nil {
		return nil, err
	}
	switch obj["group_by"] {
	case nil:
	case "space":
		s.by_space = true
	default:
		return nil, fmt.Errorf("spread group_by must be space")
	}
	if _, present := obj["capacity"]; present {
		capacity, err := parseNumber(obj, "capacity", 0)
		if err != nil {
			return nil, err
		}
		s.capacity = &capacity
	}
	if s.unit_score, err = parseNumber(obj, "unit_score", 0); err != nil {
		return nil, err
	}
	if s.unit_order, err = parseNumber(obj, "unit_order", 0); err != nil {
		return nil, err
	}
	if s.queue, err = parseNumber(obj, "queue", 0); err != nil {
		return nil, err
	}
	return s, parseSpreadOverflow(obj["overflow"], s)
}

func parseSpreadOverflow(raw any, s *spreadConfig) error {
	switch raw {
	case nil, "round_robin":
		s.overflow = spreadOverflowRoundRobin
	case "none":
		s.overflow = spreadOverflowNone
	default:
		return fmt.Errorf("spread \"overflow\" must be \"round_robin\" or \"none\"")
	}
	return nil
}
