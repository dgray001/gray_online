package ai

import (
	"fmt"
	"maps"
	"slices"
)

var selectKinds = []string{"units", "buildings", "resources", "spaces", "zones", "regions", "foundations", "gatherables", "attackables", "orders", "assignments"}

type selectorSpec struct {
	kind, role                 string
	source                     loopSource
	conditions                 []amount
	near                       []*nearSpec
	at                         *selectorSpec
	in_set, not_in_set, bucket string
	region_in_set              string
	in                         []inFilter
	rank                       []amount
	limit, skip                *amount
	capacity                   *amount
	initial                    *amount
	lets                       map[string]amount
	can_create, can_research   uint32
	can_build                  uint32
	join                       *joinPlan
	join_planned               bool
}

func (s *selectorSpec) impure() bool {
	amounts := slices.Concat(s.conditions, s.rank, slices.Collect(maps.Values(s.lets)))
	for _, near := range s.near {
		amounts = append(amounts, near.distance)
	}
	for _, bound := range []*amount{s.skip, s.limit, s.capacity, s.initial} {
		if bound != nil {
			amounts = append(amounts, *bound)
		}
	}
	return slices.ContainsFunc(amounts, func(a amount) bool { return a.impure })
}

func (s *selectorSpec) letNames() []string {
	if s == nil {
		return nil
	}
	names := make([]string, 0, len(s.lets))
	for name := range s.lets {
		names = append(names, name)
	}
	return names
}

func takeLetNames(obj map[string]any, kind string) ([]string, error) {
	raw, present := obj["let"]
	if !present {
		return nil, nil
	}
	lets, ok := raw.(map[string]any)
	if !ok {
		return nil, fmt.Errorf("\"let\" must be an object of named values")
	}
	names := make([]string, 0, len(lets))
	for name := range lets {
		if !validVarName(name) || hasItemField("select_"+kind, name) {
			return nil, fmt.Errorf("let name %q must be letters, digits and underscores and not a %s field", name, kind)
		}
		names = append(names, name)
	}
	return names, nil
}

type nearSpec struct {
	distance   amount
	of         *selectorSpec
	through    *selectorSpec
	coordinate *coordinateSpec
	zone_steps bool
}

type nearClause struct {
	locations []ZoneRef
	reach     map[Coordinate]int
	bound     int
	hoisted   bool
}

var selectorKeys = []string{"kind", "where", "rank", "limit", "skip", "let", "capacity", "initial_assigned"}

func parseForEachSelect(raw map[string]any, role string) (*selectorSpec, error) {
	value, present := raw["select"]
	if !present {
		return nil, nil
	}
	if raw["source"] != nil || raw["where"] != nil {
		return nil, fmt.Errorf("for_each \"select\" cannot be combined with \"source\" or \"where\"")
	}
	return parseSelectorSpec(value, role)
}

func parseSelectorSpec(raw any, role string) (*selectorSpec, error) {
	return parseSelectorSpecFor(raw, role, false)
}

func parseSelectorSpecFor(raw any, role string, assigning bool) (*selectorSpec, error) {
	obj, ok := raw.(map[string]any)
	if !ok {
		return nil, fmt.Errorf("a selector must be an object")
	}
	if _, present := obj["initial_assigned"]; present && !(assigning && role == "target") {
		return nil, fmt.Errorf("initial_assigned only applies to the pick of an assignment select")
	}
	if err := checkKeys(obj, selectorKeys...); err != nil {
		return nil, err
	}
	kind, _ := obj["kind"].(string)
	if !isSelectKind(kind) {
		return nil, fmt.Errorf("selector \"kind\" must be one of %v", selectKinds)
	}
	s := &selectorSpec{kind: kind, role: role}
	where, _ := obj["where"].(map[string]any)
	if _, present := obj["where"]; present && where == nil {
		return nil, fmt.Errorf("selector \"where\" must be an object")
	}
	for _, key := range []string{"within", "from"} {
		if _, present := where[key]; present {
			return nil, fmt.Errorf("select does not support %q; use \"near\" with an explicit selector or coordinate", key)
		}
	}
	let_names, err := takeLetNames(obj, kind)
	if err != nil {
		return nil, err
	}
	var rest map[string]any
	err = withLoopScopeExtra(role, s.scopeSource(), let_names, func() error {
		if err := s.parseLets(obj); err != nil {
			return err
		}
		rest, err = s.takeExtras(maps.Clone(where))
		if err != nil {
			return err
		}
		if s.rank, err = parseRank(obj); err != nil {
			return err
		}
		if s.initial, err = optionalAmount(obj, "initial_assigned"); err != nil {
			return err
		}
		if s.capacity, err = s.parseCapacity(obj, let_names, assigning); err != nil {
			return err
		}
		if s.capacity != nil && role != "target" {
			return fmt.Errorf("selector: capacity is only valid on pick selectors")
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	if _, explicit := rest["owner"]; !explicit && s.in_set != "" && (kind == "units" || kind == "buildings") {
		rest["owner"] = "any"
	}
	if s.source, err = loopSourceParsers[kind](rest); err != nil {
		return nil, fmt.Errorf("%s: %v", kind, err)
	}
	if s.limit, err = optionalAmount(obj, "limit"); err != nil {
		return nil, err
	}
	s.skip, err = optionalAmount(obj, "skip")
	return s, err
}

func (s *selectorSpec) parseCapacity(obj map[string]any, let_names []string, assigning bool) (capacity *amount, err error) {
	if !assigning {
		return optionalAmount(obj, "capacity")
	}
	extra := append(slices.Clone(let_names), "assigned")
	err = withLoopScopeExtra(s.role, s.scopeSource(), extra, func() (parsed error) {
		capacity, parsed = optionalAmount(obj, "capacity")
		return parsed
	})
	return capacity, err
}

func (s *selectorSpec) scopeSource() string {
	if s.role == "target" || s.role == "actor" {
		return "select_" + s.role + "_" + s.kind
	}
	return "select_" + s.kind
}

func (s *selectorSpec) parseLets(obj map[string]any) error {
	lets, _ := obj["let"].(map[string]any)
	s.lets = make(map[string]amount, len(lets))
	for name, raw := range lets {
		value, err := parseAmount(raw)
		if err != nil {
			return fmt.Errorf("let %s: %v", name, err)
		}
		s.lets[name] = value
	}
	return nil
}

func isSelectKind(kind string) bool {
	for _, k := range selectKinds {
		if k == kind {
			return true
		}
	}
	return false
}

func parseRank(obj map[string]any) ([]amount, error) {
	raw, present := obj["rank"]
	if !present {
		return nil, nil
	}
	entries, isList := raw.([]any)
	if !isList {
		entries = []any{raw}
	}
	ranks := make([]amount, 0, len(entries))
	for _, entry := range entries {
		rank, err := parseAmount(entry)
		if err != nil {
			return nil, fmt.Errorf("rank: %v", err)
		}
		ranks = append(ranks, rank)
	}
	return ranks, nil
}

func optionalAmount(obj map[string]any, key string) (*amount, error) {
	raw, present := obj[key]
	if !present {
		return nil, nil
	}
	a, err := parseAmount(raw)
	if err != nil {
		return nil, fmt.Errorf("%s: %v", key, err)
	}
	return &a, nil
}
