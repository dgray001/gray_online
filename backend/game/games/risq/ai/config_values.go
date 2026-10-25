package ai

import (
	"fmt"
	"strings"

	"github.com/dgray001/gray_online/game/games/risq/internal/defs"
)

var deprecatedCostTables = map[string]string{"unit_cost_": "unit", "building_cost_": "building", "tech_cost_": "tech"}

func (l *varNameLog) noteConfigRead(ref string) {
	table, field, _ := strings.Cut(ref, ".")
	if !defs.HasConfigField(table, field) {
		l.errors = append(l.errors, fmt.Sprintf("config(%s, %s, ...): unknown table or field", table, field))
	}
}

func deprecatedCostCall(name string) (string, bool) {
	for prefix, table := range deprecatedCostTables {
		rest, found := strings.CutPrefix(name, prefix)
		resource, id, split := strings.Cut(rest, "_")
		if found && split {
			return fmt.Sprintf("config(%s, cost_%s, %s)", table, resource, id), true
		}
	}
	return "", false
}
