package orders

import (
	"fmt"
	"slices"
	"strings"
)

func normalizeSnapshot(key string, value any) any {
	switch data := value.(type) {
	case map[string]any:
		for field, item := range data {
			data[field] = normalizeSnapshot(field, item)
		}
	case []any:
		for i, item := range data {
			data[i] = normalizeSnapshot("", item)
		}
		if slices.Contains([]string{"players", "units", "buildings", "resources", "subjects", "garrisoned_units", "planned_foundations", "available_mercenaries"}, key) {
			slices.SortFunc(data, func(a, b any) int { return strings.Compare(fmt.Sprint(a), fmt.Sprint(b)) })
		}
	}
	return value
}
