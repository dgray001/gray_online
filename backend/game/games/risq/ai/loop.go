package ai

import (
	"fmt"
	"slices"
	"strings"
)

// Most loop bodies run in one decision, across every loop, so a nested loop over a big source cannot stall a turn
const maxLoopSteps = 10000

type forEachAction struct {
	name          string
	source        loopSource
	select_source *selectorSpec
	max           amount
	body          []Action
}

func (a *forEachAction) ToOrders(view View, internals *Internals) []Order {
	previous := internals.eval_errors
	limit := a.max.int(view, internals)
	if internals.eval_errors != previous {
		return nil
	}
	var items []loopItem
	if a.select_source != nil {
		for _, item := range a.resolveWithSelectionCensus(view, internals) {
			items = append(items, item)
		}
	} else {
		items = a.source.items(view, internals)
	}
	orders := make([]Order, 0)
	for n, item := range items {
		if (limit > 0 && n >= limit) || !internals.takeLoopStep() {
			break
		}
		restore := internals.bindItem(a.name, item)
		orders = append(orders, runActions(a.body, view, internals)...)
		restore()
	}
	return orders
}

func (a *forEachAction) resolveWithSelectionCensus(view View, internals *Internals) []selItem {
	census, previous := newCensus(view, internals), internals.select_census
	internals.select_census = census
	defer func() { internals.select_census = previous }()
	return a.select_source.resolve(view, internals, census)
}

// Makes the item visible to var(name.field) until the returned function puts back whatever name meant before
func (i *Internals) bindItem(name string, item loopItem) func() {
	if i.loop_items == nil {
		i.loop_items = make(map[string]loopItem)
	}
	previous, had := i.loop_items[name]
	i.loop_items[name] = item
	i.binding_depth++
	return func() {
		i.binding_depth--
		if had {
			i.loop_items[name] = previous
		} else {
			delete(i.loop_items, name)
		}
	}
}

func (i *Internals) takeLoopStep() bool {
	i.loop_steps++
	if i.loop_steps > maxLoopSteps {
		i.warnOnce(fmt.Sprintf("loop_steps:%d", i.rule_index), fmt.Sprintf("ai: rule %d: for_each stopped after %d steps", i.rule_index, maxLoopSteps))
		return false
	}
	return true
}

func parseForEach(raw map[string]any) (Action, error) {
	if err := checkActionKeys(raw, "source", "select", "as", "where", "max", "do"); err != nil {
		return nil, fmt.Errorf("for_each: %w", err)
	}
	if _, err := parseNumber(raw, "max", 0); err != nil {
		return nil, fmt.Errorf("for_each: %v", err)
	}
	if value, present := raw["where"]; present {
		if _, ok := value.(map[string]any); !ok {
			return nil, fmt.Errorf("\"where\" must be an object")
		}
	}
	name, _ := raw["as"].(string)
	if !validVarName(name) {
		return nil, fmt.Errorf("for_each requires an \"as\" name of letters, digits and underscores")
	}
	select_source, err := parseForEachSelect(raw, name)
	if err != nil {
		return nil, err
	}
	if select_source != nil {
		if slices.Contains([]string{"actor", "target", "item", "anchor", "place"}, name) {
			return nil, fmt.Errorf("for_each select cannot name its item %q, which selectors reserve for their own roles", name)
		}
		action := &forEachAction{name: name, select_source: select_source, max: parseMax(raw)}
		err = withLoopScopeExtra(name, "select_"+select_source.kind, select_source.letNames(), func() error {
			action.body, err = parseNested("do", raw["do"])
			return err
		})
		return action, err
	}
	source_name, _ := raw["source"].(string)
	if source_name == "orders" || source_name == "assignments" {
		selection := map[string]any{"kind": source_name}
		if where, present := raw["where"]; present {
			selection["where"] = where
		}
		converted := map[string]any{"action": "for_each", "as": raw["as"], "do": raw["do"], "select": selection}
		if limit, present := raw["max"]; present {
			converted["max"] = limit
		}
		return parseForEach(converted)
	}
	parse_source, known := loopSourceParsers[source_name]
	if !known {
		return nil, fmt.Errorf("for_each \"source\" must be players, spaces, regions, mercenaries, units, buildings, resources or zones")
	}
	where, _ := raw["where"].(map[string]any)
	source, err := parse_source(where)
	if err != nil {
		return nil, fmt.Errorf("for_each %s: %v", source_name, err)
	}
	action := &forEachAction{name: name, source: source, max: parseMax(raw)}
	err = withLoopScope(name, source_name, func() error {
		var body_err error
		action.body, body_err = parseNested("do", raw["do"])
		return body_err
	})
	return action, err
}

// distance(a, b) between the items bound under two names, in space steps or zone steps
func (i *Internals) distance(view View, a string, b string, zone_steps bool) (float64, error) {
	from, _, from_located := locOf(i.loop_items[a])
	to, _, to_located := locOf(i.loop_items[b])
	if !from_located || !to_located {
		return 0, fmt.Errorf("distance(%s, %s): both names must be bound to located items", a, b)
	}
	if zone_steps {
		return float64(view.LocationDistance(from, to)), nil
	}
	return float64(view.SpaceDistance(from.Space, to.Space)), nil
}

// var(name.field) while a loop named name is visiting an item; 0 (with one warning) when no loop or field matches
func (i *Internals) itemField(name string) float64 {
	prefix, field, _ := strings.Cut(name, ".")
	if item := i.loop_items[prefix]; item != nil {
		previous := i.eval_errors
		if value, ok := item.field(field); ok {
			return value
		}
		if i.eval_errors != previous {
			return 0
		}
	}
	i.warnOnce("item:"+name, fmt.Sprintf("ai: var(%s) does not name a field of an item being visited", name))
	return 0
}
