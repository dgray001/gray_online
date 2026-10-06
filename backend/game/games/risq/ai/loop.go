package ai

import (
	"fmt"
	"strings"
)

// Most loop bodies run in one decision, across every loop, so a nested loop over a big source cannot stall a turn
const maxLoopSteps = 10000

type forEachAction struct {
	name   string
	source loopSource
	max    amount
	body   []Action
}

func (a *forEachAction) ToOrders(view View, internals *Internals) []Order {
	limit := a.max.int(view, internals)
	orders := make([]Order, 0)
	for n, item := range a.source.items(view, internals) {
		if (limit > 0 && n >= limit) || !internals.takeLoopStep() {
			break
		}
		restore := internals.bindItem(a.name, item)
		orders = append(orders, runActions(a.body, view, internals)...)
		restore()
	}
	return orders
}

// Makes the item visible to var(name.field) until the returned function puts back whatever name meant before
func (i *Internals) bindItem(name string, item loopItem) func() {
	if i.loop_items == nil {
		i.loop_items = make(map[string]loopItem)
	}
	previous, had := i.loop_items[name]
	i.loop_items[name] = item
	return func() {
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
		i.warnOnce("loop_steps", fmt.Sprintf("ai: for_each stopped after %d steps in one decision", maxLoopSteps))
		return false
	}
	return true
}

func parseForEach(raw map[string]any) (Action, error) {
	source_name, _ := raw["source"].(string)
	parse_source, known := loopSourceParsers[source_name]
	if !known {
		return nil, fmt.Errorf("for_each \"source\" must be players, spaces, units, buildings, resources or zones")
	}
	name, _ := raw["as"].(string)
	if !validVarName(name) {
		return nil, fmt.Errorf("for_each requires an \"as\" name of letters, digits and underscores")
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

// var(name.field) while a loop named name is visiting an item; 0 (with one warning) when no loop or field matches
func (i *Internals) itemField(name string) float64 {
	prefix, field, _ := strings.Cut(name, ".")
	if item := i.loop_items[prefix]; item != nil {
		if value, ok := item.field(field); ok {
			return value
		}
	}
	i.warnOnce("item:"+name, fmt.Sprintf("ai: var(%s) does not name a field of an item being visited", name))
	return 0
}
