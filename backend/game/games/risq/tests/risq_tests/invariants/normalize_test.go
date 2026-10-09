package invariants

import (
	"encoding/json"
	"slices"
	"strings"
)

func canonical(value any) string {
	data, err := json.Marshal(value)
	if err != nil {
		panic(err)
	}
	return string(data)
}

type canonicalItem struct {
	value any
	key   string
}

func unordered(key string, data []any) bool {
	if key == "spaces" {
		return len(data) > 0 && isNumber(data[0])
	}
	return slices.Contains([]string{"players", "units", "buildings", "resources", "subjects", "garrisoned_units", "planned_foundations", "available_mercenaries", "regions", "scores", "units_created", "buildings_built", "techs_researched", "combat", "failures"}, key)
}

func isNumber(value any) bool { _, ok := value.(float64); return ok }

func normalize(key string, value any) any {
	switch data := value.(type) {
	case object:
		delete(data, "game_base")
		if key == "player" {
			delete(data, "client_id")
			delete(data, "nickname")
		}
		for field, item := range data {
			data[field] = normalize(field, item)
		}
	case []any:
		for i, item := range data {
			data[i] = normalize("", item)
		}
		if unordered(key, data) {
			items := make([]canonicalItem, len(data))
			for i, item := range data {
				items[i] = canonicalItem{value: item, key: canonical(item)}
			}
			slices.SortFunc(items, func(a, b canonicalItem) int { return strings.Compare(a.key, b.key) })
			for i, item := range items {
				data[i] = item.value
			}
		}
	}
	return value
}
