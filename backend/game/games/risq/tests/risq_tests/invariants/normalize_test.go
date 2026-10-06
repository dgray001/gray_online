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
			slices.SortFunc(data, func(a, b any) int { return strings.Compare(canonical(a), canonical(b)) })
		}
	}
	return value
}
